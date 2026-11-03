package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/slinxlink/node/internal/database"
)

func TestDynamicAllocation(t *testing.T) {
	ep := database.Endpoint{Type: "socks", Username: "customer", Password: "secret", Dynamic: `{"mode":"session","ttl_seconds":120,"username_template":"{username}-{session}"}`}
	first, err := RotateDynamic(ep)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RotateDynamic(first)
	if err != nil {
		t.Fatal(err)
	}
	u, _, err := DynamicCredentials(first)
	if err != nil || u == "customer" {
		t.Fatal("template not applied")
	}
	p, _ := ParseDynamic(first)
	q, _ := ParseDynamic(second)
	if p.SessionID == q.SessionID || time.Until(p.ExpiresAt) < time.Minute {
		t.Fatal("session allocation")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("X-Test") != "token" {
			t.Error("provider request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"host":"127.0.0.1","port":18080,"user":"new-user"}}`))
	}))
	defer server.Close()
	ep.Dynamic = `{"mode":"api","url":"` + server.URL + `","method":"POST","headers":{"X-Test":"token"},"mapping":{"server":"data.host","server_port":"data.port","username":"data.user"}}`
	got, err := RotateDynamic(ep)
	if err != nil || got.ServerPort != 18080 || got.Username != "new-user" {
		t.Fatalf("provider mapping: %v", err)
	}
}
