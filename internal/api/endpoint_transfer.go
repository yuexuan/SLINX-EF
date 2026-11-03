package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
	"gorm.io/gorm"
)

func ExportEndpoints(c *gin.Context) {
	var req struct {
		Tags   []string `json:"tags"`
		Format string   `json:"format"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"error": "请求无效"})
		return
	}
	var nodes []database.Endpoint
	database.DB.Find(&nodes)
	content, err := core.ExportNodes(nodes, req.Tags, req.Format)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": content})
}

func ImportEndpoints(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var req struct {
		Content string `json:"content"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.Content) > 4<<20 {
		c.JSON(400, gin.H{"error": "导入内容无效或超过 4MB"})
		return
	}
	imported, err := core.ImportNodes(req.Content)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	var nodes []database.Endpoint
	database.DB.Find(&nodes)
	for i := range imported {
		if msg := util.ValidateTag(imported[i].Tag); msg != "" {
			c.JSON(400, gin.H{"error": msg})
			return
		}
		imported[i].ID = 0
		imported[i].CreatedAt = time.Now()
		imported[i].UpdatedAt = time.Now()
	}
	merged := append(nodes, imported...)
	if err := core.ValidateGraph(merged); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	for i := range merged {
		merged[i].Enable = true
	}
	for _, n := range imported {
		cfg, e := core.NodeConfig(merged, n.Tag)
		if e == nil {
			e = core.CheckConfig(cfg)
		}
		if e != nil {
			c.JSON(400, gin.H{"error": fmt.Sprintf("%s: %v", n.Tag, e)})
			return
		}
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for i := range imported {
			if e := tx.Create(&imported[i]).Error; e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err = core.Default.Apply(); err != nil {
		for _, n := range imported {
			database.DB.Delete(&n)
		}
		c.JSON(500, gin.H{"error": "导入未应用，已恢复: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"count": len(imported), "applied": true})
}
