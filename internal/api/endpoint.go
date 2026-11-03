package api

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/route"
	"github.com/slinxlink/node/internal/service"
	"github.com/slinxlink/node/internal/util"
)

var endpointMu sync.Mutex

func GetEndpoints(c *gin.Context) {
	var list []database.Endpoint
	database.DB.Find(&list)
	items := make([]gin.H, 0, len(list))
	for _, ep := range list {
		items = append(items, gin.H{"ID": ep.ID, "Tag": ep.Tag, "Enable": ep.Enable, "Type": ep.Type, "Server": ep.Server, "ServerPort": ep.ServerPort, "PeerAddress": ep.PeerAddress, "PeerPort": ep.PeerPort, "Detour": ep.Detour})
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func SaveEndpoint(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var endpoint database.Endpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endpoint.Tag = strings.TrimSpace(endpoint.Tag)
	policy, policyErr := service.ParseDynamic(endpoint)
	if policyErr != nil {
		c.JSON(400, gin.H{"error": policyErr.Error()})
		return
	}
	if policy.Mode == "session" && policy.SessionID == "" {
		var err error
		endpoint, err = service.RotateDynamic(endpoint)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}

	if msg := util.ValidateTag(endpoint.Tag); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	var existing database.Endpoint
	if err := database.DB.Where("tag = ? AND id != ?", endpoint.Tag, endpoint.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该标签已存在"})
		return
	}

	if endpoint.Tag == "direct" || endpoint.Tag == "block" {
		c.JSON(400, gin.H{"error": "保留标签"})
		return
	}
	if endpoint.Type != "wireguard" {
		if _, err := core.BuildOutbound(endpoint); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}
	if endpoint.Type == "wireguard" {
		var address []string
		if err := json.Unmarshal([]byte(endpoint.Address), &address); err != nil || len(address) == 0 {
			c.JSON(400, gin.H{"error": "需要至少一个 WireGuard 本地 CIDR"})
			return
		}
		for _, cidr := range address {
			if _, err := netip.ParsePrefix(cidr); err != nil {
				c.JSON(400, gin.H{"error": "WireGuard CIDR 无效"})
				return
			}
		}
		if endpoint.MTU < 576 || endpoint.MTU > 9000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MTU 必须在 576 - 9000 之间"})
			return
		}
		var reserved []int
		if err := json.Unmarshal([]byte(endpoint.Reserved), &reserved); err != nil || (len(reserved) != 0 && len(reserved) != 3) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reserved 格式错误，需为3个数字"})
			return
		}
		for _, v := range reserved {
			if v < 0 || v > 255 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Reserved 数值必须在 0-255 之间"})
				return
			}
		}
		if endpoint.PrivateKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写密钥对"})
			return
		}
		if endpoint.PeerAddress == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写对端地址"})
			return
		}
		if endpoint.PeerPort < 1 || endpoint.PeerPort > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "对端端口必须在 1 - 65535 之间"})
			return
		}
		if endpoint.PeerPublicKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写对端公钥"})
			return
		}
	}

	var nodes []database.Endpoint
	database.DB.Find(&nodes)
	var old database.Endpoint
	if endpoint.ID != 0 {
		if err := database.DB.First(&old, endpoint.ID).Error; err != nil {
			c.JSON(404, gin.H{"error": "节点不存在"})
			return
		}
	}
	for i := range nodes {
		if nodes[i].ID == endpoint.ID {
			nodes[i] = endpoint
		}
	}
	if endpoint.ID == 0 {
		nodes = append(nodes, endpoint)
	}
	if err := core.ValidateGraph(nodes); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// Validate disabled nodes too, without changing persisted enable state.
	for i := range nodes {
		nodes[i].Enable = true
	}
	cfg, err := core.NodeConfig(nodes, endpoint.Tag)
	if err == nil {
		err = core.CheckConfig(cfg)
	}
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if old.ID != 0 && old.Tag != endpoint.Tag {
		var refs int64
		database.DB.Model(&database.Endpoint{}).Where("detour = ?", old.Tag).Count(&refs)
		if refs > 0 {
			c.JSON(400, gin.H{"error": "标签被中继引用，请先解除引用"})
			return
		}
	}
	if err := database.DB.Save(&endpoint).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if old.ID != 0 && old.Tag != endpoint.Tag {
		route.CleanupRule("endpoint", old.Tag, endpoint.Tag)
	}
	if err := core.Default.Apply(); err != nil {
		if old.ID == 0 {
			database.DB.Delete(&endpoint)
		} else {
			database.DB.Save(&old)
			if old.Tag != endpoint.Tag {
				route.CleanupRule("endpoint", endpoint.Tag, old.Tag)
			}
		}
		c.JSON(500, gin.H{"error": "配置未应用，已恢复节点: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": endpoint, "applied": core.Default.Status() == "running", "core_status": core.Default.Status()})
}

func DeleteEndpoint(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var endpoint database.Endpoint
	if err := database.DB.First(&endpoint, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "节点不存在"})
		return
	}
	// Preserve dependent nodes and route bindings as disabled, never silently direct.
	var dependents []database.Endpoint
	database.DB.Where("detour = ?", endpoint.Tag).Find(&dependents)
	for _, n := range dependents {
		n.Enable = false
		n.Detour = ""
		database.DB.Save(&n)
	}
	var rules []database.Rule
	database.DB.Find(&rules)
	route.CleanupRule("endpoint", endpoint.Tag, "block")
	if err := database.DB.Delete(&endpoint).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := core.Default.Apply(); err != nil {
		database.DB.Save(&endpoint)
		for _, n := range dependents {
			database.DB.Save(&n)
		}
		database.DB.Where("1=1").Delete(&database.Rule{})
		for _, r := range rules {
			database.DB.Save(&r)
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"applied": core.Default.Status() == "running", "core_status": core.Default.Status()})
}

func ToggleEndpoint(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var endpoint database.Endpoint
	if err := database.DB.First(&endpoint, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "节点不存在"})
		return
	}
	endpoint.Enable = !endpoint.Enable
	if err := database.DB.Save(&endpoint).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := core.Default.Apply(); err != nil {
		endpoint.Enable = !endpoint.Enable
		database.DB.Save(&endpoint)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": endpoint, "applied": core.Default.Status() == "running", "core_status": core.Default.Status()})
}

func CreateWarpEndpoint(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var req struct {
		Address       string
		Reserved      string
		PrivateKey    string
		PublicKey     string
		PeerAddress   string
		PeerPort      int
		PeerPublicKey string
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var endpoint database.Endpoint
	database.DB.Where("tag = ?", "warp").First(&endpoint)

	if endpoint.ID == 0 {
		endpoint.Enable = true
		endpoint.Tag = "warp"
		endpoint.Type = "wireguard"
		endpoint.MTU = 1408
		endpoint.AllowedIPs = "0.0.0.0/0,::/0"
	}

	endpoint.Address = req.Address
	endpoint.Reserved = req.Reserved
	endpoint.PrivateKey = req.PrivateKey
	endpoint.PublicKey = req.PublicKey
	endpoint.PeerAddress = req.PeerAddress
	endpoint.PeerPort = req.PeerPort
	endpoint.PeerPublicKey = req.PeerPublicKey

	if err := database.DB.Save(&endpoint).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	util.Info("[endpoint] 创建/更新 WARP 端点")
	if err := core.Default.Apply(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": endpoint})
}

func GetEndpoint(c *gin.Context) {
	var ep database.Endpoint
	if err := database.DB.First(&ep, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "节点不存在"})
		return
	}
	c.JSON(200, gin.H{"data": ep})
}
