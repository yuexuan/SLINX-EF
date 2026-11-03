package api

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/service"
)

func probeIP(id uint) (string, error) {
	d, close, err := rawNodeDetector(id)
	if err != nil {
		return "", err
	}
	defer close()
	return d.PublicIP()
}

func rotateNode(id uint) (database.NodeEvent, error) {
	var old database.Endpoint
	event := database.NodeEvent{NodeID: id, Kind: "rotation", CreatedAt: time.Now()}
	if err := database.DB.First(&old, id).Error; err != nil {
		return event, err
	}
	event.NodeTag = old.Tag
	event.BeforeIP, _ = probeIP(id)
	next, err := service.RotateDynamic(old)
	if err != nil {
		event.Status = "供应商分配失败"
		database.DB.Create(&event)
		return event, err
	}
	var nodes []database.Endpoint
	database.DB.Find(&nodes)
	for i := range nodes {
		if nodes[i].ID == id {
			nodes[i] = next
		}
	}
	cfg, err := core.NodeConfig(nodes, next.Tag)
	if err == nil {
		err = core.CheckConfig(cfg)
	}
	if err == nil {
		err = database.DB.Save(&next).Error
	}
	if err == nil {
		err = core.Default.Apply()
	}
	if err != nil {
		database.DB.Save(&old)
		event.Status = "配置应用失败，已恢复"
		database.DB.Create(&event)
		return event, err
	}
	event.AfterIP, err = probeIP(id)
	if err != nil {
		event.Status = "已应用，真实出口未验证"
	} else if event.BeforeIP == event.AfterIP {
		event.Status = "已应用，实际 IP 未变化"
	} else {
		event.Status = "出口 IP 已验证"
	}
	database.DB.Create(&event)
	return event, err
}

func RotateEndpoint(c *gin.Context) {
	var ep database.Endpoint
	if err := database.DB.First(&ep, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"error": "节点不存在"})
		return
	}
	event, err := rotateNode(ep.ID)
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error(), "data": event})
		return
	}
	c.JSON(200, gin.H{"data": event})
}

func nodeDetector(id uint) (*service.Detector, func(), error) {
	if id != 0 {
		var ep database.Endpoint
		if err := database.DB.First(&ep, id).Error; err != nil {
			return nil, nil, err
		}
		p, err := service.ParseDynamic(ep)
		if err != nil {
			return nil, nil, err
		}
		if p.Mode == "session" && !p.ExpiresAt.IsZero() && time.Now().After(p.ExpiresAt) {
			if _, err = rotateNode(id); err != nil {
				return nil, nil, err
			}
		} else if p.RotateOnFailure {
			_, err = probeIP(id)
			// Only transport failures qualify. HTTP 403/429 or invalid target responses never do.
			var networkErr net.Error
			if err != nil && errors.As(err, &networkErr) {
				if _, retryErr := probeIP(id); retryErr != nil {
					if _, err = rotateNode(id); err != nil {
						return nil, nil, fmt.Errorf("按需轮换失败: %w", err)
					}
				}
			}
		}
	}
	return rawNodeDetector(id)
}
