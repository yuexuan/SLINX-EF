package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func endpointTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	binary := os.Getenv("EF_CORE_PATH")
	if binary == "" {
		t.Skip("full pinned core not configured")
	}
	previousDB, previousCore := database.DB, core.Default
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := database.DB.DB()
	sqlDB.SetMaxOpenConns(1)
	database.DB.AutoMigrate(&database.Config{}, &database.Core{}, &database.Endpoint{}, &database.Inbound{}, &database.User{}, &database.Rule{}, &database.IP{}, &database.Unlock{}, &database.NodeEvent{})
	database.DB.Create(&database.Core{LogLevel: "warn"})
	database.DB.Create(&database.Config{})
	core.Default = &core.Manager{BinPath: binary, ConfigPath: filepath.Join(t.TempDir(), "sing-box.json")}
	t.Cleanup(func() { sqlDB.Close(); database.DB = previousDB; core.Default = previousCore })
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/endpoint/save", SaveEndpoint)
	r.GET("/endpoint", GetEndpoints)
	r.GET("/endpoint/:id", GetEndpoint)
	r.PUT("/endpoint/:id/toggle", ToggleEndpoint)
	r.DELETE("/endpoint/:id", DeleteEndpoint)
	r.POST("/endpoint/export", ExportEndpoints)
	r.POST("/endpoint/import", ImportEndpoints)
	r.PUT("/rule", SaveRule)
	r.POST("/detect/ip/fetch", FetchIP)
	r.GET("/detect/ip", DetectIP)
	r.POST("/detect/unlock/fetch", FetchUnlock)
	return r
}

func apiRequest(r *gin.Engine, method, path string, value any) (int, map[string]any) {
	b, _ := json.Marshal(value)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	return w.Code, result
}

func TestManagedNodeLifecycleAndFailClosed(t *testing.T) {
	r := endpointTestServer(t)
	relay := database.Endpoint{Enable: true, Tag: "relay", Type: "socks", Server: "127.0.0.1", ServerPort: 19876, Username: "secret-user", Password: "secret-password"}
	if status, res := apiRequest(r, "PUT", "/endpoint/save", relay); status != 200 {
		t.Fatalf("save: %v", res)
	}
	var saved database.Endpoint
	database.DB.Where("tag = ?", "relay").First(&saved)
	exit := database.Endpoint{Enable: true, Tag: "exit", Type: "ssh", Server: "127.0.0.1", ServerPort: 22, Username: "root", Password: "secret-password", Detour: "relay"}
	if status, res := apiRequest(r, "PUT", "/endpoint/save", exit); status != 200 {
		t.Fatalf("SSH save: %v", res)
	}
	_, list := apiRequest(r, "GET", "/endpoint", nil)
	raw, _ := json.Marshal(list)
	if strings.Contains(string(raw), "secret-user") || strings.Contains(string(raw), "secret-password") {
		t.Fatal("credentials leaked into list")
	}
	database.DB.Create(&database.Inbound{Enable: true, Port: 443})
	if status, res := apiRequest(r, "PUT", "/rule", map[string]any{"tag": "exit", "inbounds": []string{"443"}}); status != 200 {
		t.Fatalf("bind: %v", res)
	}
	assertRoute := func(target string) {
		t.Helper()
		data, _ := os.ReadFile(core.Default.ConfigPath)
		var cfg map[string]any
		json.Unmarshal(data, &cfg)
		rules := cfg["route"].(map[string]any)["rules"].([]any)
		if rules[1].(map[string]any)["outbound"] != target {
			t.Fatalf("route did not fail closed: %s", data)
		}
	}
	assertRoute("exit")
	id := jsonNumber(saved.ID)
	if status, res := apiRequest(r, "PUT", "/endpoint/"+id+"/toggle", nil); status != 200 {
		t.Fatal(res)
	}
	assertRoute("block")
	if status, res := apiRequest(r, "PUT", "/endpoint/"+id+"/toggle", nil); status != 200 {
		t.Fatal(res)
	}
	assertRoute("exit")
	saved.Detour = "exit"
	if status, _ := apiRequest(r, "PUT", "/endpoint/save", saved); status != 400 {
		t.Fatal("cycle accepted")
	}
	if status, res := apiRequest(r, "DELETE", "/endpoint/"+id, nil); status != 200 {
		t.Fatal(res)
	}
	assertRoute("block")
	var after database.Endpoint
	database.DB.Where("tag = ?", "exit").First(&after)
	if after.Enable || after.Detour != "" {
		t.Fatal("dependent not blocked")
	}
	// Missing or malformed selected nodes must not silently select server-direct.
	if status, res := apiRequest(r, "POST", "/detect/ip/fetch", map[string]any{"source": "ippure.com", "node_id": 9999}); status == 200 || res["data"] != nil {
		t.Fatal("missing node fell back to direct")
	}
	if status, _ := apiRequest(r, "POST", "/detect/ip/fetch", map[string]any{"node_id": "invalid"}); status != 400 {
		t.Fatal("malformed node fell back to direct")
	}
}

func jsonNumber(id uint) string { b, _ := json.Marshal(id); return string(b) }

func TestNodeImportAtomicity(t *testing.T) {
	r := endpointTestServer(t)
	content := `{"format":"slinx-ef/nodes-v1","nodes":[{"Enable":true,"Tag":"imported","Type":"http","Server":"127.0.0.1","ServerPort":1080}]}`
	if status, res := apiRequest(r, "POST", "/endpoint/import", map[string]any{"content": content}); status != 200 {
		t.Fatal(res)
	}
	if status, _ := apiRequest(r, "POST", "/endpoint/import", map[string]any{"content": content}); status != 400 {
		t.Fatal("duplicate import accepted")
	}
	var count int64
	database.DB.Model(&database.Endpoint{}).Count(&count)
	if count != 1 {
		t.Fatal("failed import changed nodes")
	}
	bad := strings.Replace(content, `"http"`, `"unsupported"`, 1)
	bad = strings.Replace(bad, `"imported"`, `"invalid"`, 1)
	if status, _ := apiRequest(r, "POST", "/endpoint/import", map[string]any{"content": bad}); status != 400 {
		t.Fatal("invalid protocol accepted")
	}
	database.DB.Model(&database.Endpoint{}).Count(&count)
	if count != 1 {
		t.Fatal("invalid import was persisted")
	}
}
