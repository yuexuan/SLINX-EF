package api

import (
	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/service"
	"github.com/slinxlink/node/internal/util"
	"net/http"
	"strconv"
	"time"
)

// GET /detect/ip 读库返回
func DetectIP(c *gin.Context) {
	var records []database.IP
	database.DB.Where("node_id = ?", c.DefaultQuery("node_id", "0")).Order("id desc").Limit(500).Find(&records)
	c.JSON(200, gin.H{"data": records})
}

// POST /detect/ip/fetch 触发查询
func FetchIP(c *gin.Context) {
	var req struct {
		Source string `json:"source"`
		NodeID uint   `json:"node_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "检测请求无效"})
		return
	}
	if req.Source == "" {
		req.Source = "ip-api.com"
	}

	util.Info("[detect] 查询 IP 信息: %s", req.Source)
	detector, close, err := nodeDetector(req.NodeID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	defer close()
	record, err := detector.FetchIPInfo(req.Source)
	if err != nil {
		database.DB.Create(&database.NodeEvent{NodeID: req.NodeID, Kind: "detection", Status: "检测失败（未回退直连）", CreatedAt: time.Now()})
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": record})
}

// GET /detect/unlock 读库返回
func DetectUnlock(c *gin.Context) {
	var records []database.Unlock
	database.DB.Where("node_id = ?", c.DefaultQuery("node_id", "0")).Order("id desc").Limit(500).Find(&records)
	c.JSON(200, gin.H{"data": records})
}

// POST /detect/unlock/fetch 触发查询
func FetchUnlock(c *gin.Context) {
	util.Info("[detect] 查询流媒体解锁")
	var req struct {
		NodeID uint `json:"node_id"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "检测请求无效"})
		return
	}
	detector, close, err := nodeDetector(req.NodeID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	defer close()
	records, err := detector.FetchUnlockInfo()
	if err != nil {
		database.DB.Create(&database.NodeEvent{NodeID: req.NodeID, Kind: "detection", Status: "检测失败（未回退直连）", CreatedAt: time.Now()})
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": records})
}

// GET /detect/back-route 读库返回
func DetectBackRoute(c *gin.Context) {
	records, err := service.GetBackRouteInfo()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": records})
}

// POST /detect/back-route/fetch 触发查询
func FetchBackRoute(c *gin.Context) {
	util.Info("[detect] 查询回程路由")
	records, err := service.FetchBackRouteInfo()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": records})
}

func rawNodeDetector(id uint) (*service.Detector, func(), error) {
	if id == 0 {
		client := service.DirectClient()
		return service.NewDetector(client, 0, "服务器直连"), func() { client.CloseIdleConnections() }, nil
	}
	var ep database.Endpoint
	if err := database.DB.First(&ep, id).Error; err != nil {
		return nil, nil, err
	}
	var nodes []database.Endpoint
	database.DB.Find(&nodes)
	cfg, err := core.NodeConfig(nodes, ep.Tag)
	if err != nil {
		return nil, nil, err
	}
	client, close, err := core.ProbeClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	return service.NewDetector(client, id, ep.Tag), close, nil
}

func NodeHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.DefaultQuery("node_id", "0"), 10, 64)
	var events []database.NodeEvent
	database.DB.Where("node_id = ?", id).Order("id desc").Limit(100).Find(&events)
	c.JSON(http.StatusOK, gin.H{"data": events})
}
