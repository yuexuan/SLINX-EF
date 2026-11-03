package api

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/core"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

// GET /rule 整表返回
func GetRule(c *gin.Context) {
	var rules []database.Rule
	database.DB.Order("sort asc, `index` asc").Find(&rules)
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

func SaveRule(c *gin.Context) {
	endpointMu.Lock()
	defer endpointMu.Unlock()
	var req struct {
		Tag      string   `json:"tag"`
		Inbounds []string `json:"inbounds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Tag != "direct" && req.Tag != "block" {
		var count int64
		database.DB.Model(&database.Endpoint{}).Where("tag = ?", req.Tag).Count(&count)
		if count == 0 {
			c.JSON(400, gin.H{"error": "路由节点不存在"})
			return
		}
	}
	var before []database.Rule
	database.DB.Find(&before)
	var inbounds []database.Inbound
	database.DB.Find(&inbounds)
	for _, tag := range req.Inbounds {
		found := false
		for _, ib := range inbounds {
			if fmt.Sprint(ib.Port) == tag {
				found = true
				break
			}
		}
		if !found {
			c.JSON(400, gin.H{"error": "入站不存在"})
			return
		}
	}
	// One managed inbound has one effective exit. Move selected bindings from other groups.
	for _, row := range before {
		if row.Key != "inbound" {
			continue
		}
		same := false
		for _, out := range before {
			if out.Sort == row.Sort && out.Key == "outbound" {
				var tag string
				json.Unmarshal([]byte(out.Value), &tag)
				same = tag == req.Tag
			}
		}
		if same {
			continue
		}
		var tags []string
		json.Unmarshal([]byte(row.Value), &tags)
		remaining := []string{}
		for _, tag := range tags {
			selected := false
			for _, s := range req.Inbounds {
				if s == tag {
					selected = true
					break
				}
			}
			if !selected {
				remaining = append(remaining, tag)
			}
		}
		if len(remaining) != len(tags) {
			if len(remaining) == 0 {
				database.DB.Where("sort = ?", row.Sort).Delete(&database.Rule{})
			} else {
				b, _ := json.Marshal(remaining)
				database.DB.Model(&row).Update("value", string(b))
			}
		}
	}
	tagJSON, _ := json.Marshal(req.Tag)
	inboundJSON, _ := json.Marshal(req.Inbounds)

	var outboundRow database.Rule
	err := database.DB.Where("`key` = ? AND value = ?", "outbound", string(tagJSON)).First(&outboundRow).Error

	if err == nil {
		// 已存在对应 tag 的组
		if len(req.Inbounds) == 0 {
			// 清空了，直接删掉整组
			database.DB.Where("sort = ?", outboundRow.Sort).Delete(&database.Rule{})
		} else {
			database.DB.Model(&database.Rule{}).
				Where("sort = ? AND `key` = ?", outboundRow.Sort, "inbound").
				Update("value", string(inboundJSON))
		}
	} else if len(req.Inbounds) > 0 {
		// 不存在,且这次有值才新开一组（空列表不需要新建）
		var maxSort int
		database.DB.Model(&database.Rule{}).Select("COALESCE(MAX(sort),0)").Scan(&maxSort)
		newSort := maxSort + 1

		database.DB.Create(&database.Rule{Sort: newSort, Index: 0, Key: "inbound", Value: string(inboundJSON)})
		database.DB.Create(&database.Rule{Sort: newSort, Index: 1, Key: "outbound", Value: string(tagJSON)})
	}

	util.Info("[rule] 更新路由: %s", req.Tag)
	if err := core.Default.Apply(); err != nil {
		database.DB.Transaction(func(tx *gorm.DB) error {
			if e := tx.Where("1=1").Delete(&database.Rule{}).Error; e != nil {
				return e
			}
			for _, r := range before {
				if e := tx.Create(&r).Error; e != nil {
					return e
				}
			}
			return nil
		})
		c.JSON(500, gin.H{"error": "路由应用失败，已恢复: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "core_status": core.Default.Status()})
}
