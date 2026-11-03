package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpgradeLegacyDetectionIndexesPreservesHistory(t *testing.T) {
	previous := DB
	defer func() { DB = previous }()
	var err error
	DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	DB.AutoMigrate(&IP{}, &Unlock{})
	DB.Exec("CREATE UNIQUE INDEX idx_ip_source_version ON ip(source,ip_version)")
	DB.Exec("CREATE UNIQUE INDEX idx_unlock_ip_version_platform ON unlock(ip,ip_version,platform)")
	first := IP{Source: "ippure.com", IPVersion: "v4", IP: "203.0.113.1"}
	DB.Create(&first)
	unlock := Unlock{IP: "203.0.113.1", IPVersion: "v4", Platform: "chatgpt", Status: "true"}
	DB.Create(&unlock)
	DB.Exec("UPDATE ip SET node_id=NULL WHERE id=?", first.ID)
	DB.Exec("UPDATE unlock SET node_id=NULL WHERE id=?", unlock.ID)
	if err = migrateNodeHistory(); err != nil {
		t.Fatal(err)
	}
	if err = migrateNodeHistory(); err != nil {
		t.Fatal("non-idempotent migration:", err)
	}
	for _, node := range []uint{1, 2, 1} {
		if err = DB.Create(&IP{NodeID: node, Source: "ippure.com", IPVersion: "v4", IP: "203.0.113.1"}).Error; err != nil {
			t.Fatal(err)
		}
		if err = DB.Create(&Unlock{NodeID: node, IP: "203.0.113.1", IPVersion: "v4", Platform: "chatgpt"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var legacy IP
	if err = DB.Where("node_id = ? AND id = ?", 0, first.ID).First(&legacy).Error; err != nil {
		t.Fatal("legacy result missing:", err)
	}
	if legacy.IP != first.IP || legacy.NodeTag != "服务器直连" {
		t.Fatal("legacy data changed")
	}
	var count int64
	DB.Model(&IP{}).Count(&count)
	if count != 4 {
		t.Fatal("IP history lost")
	}
	DB.Model(&Unlock{}).Count(&count)
	if count != 4 {
		t.Fatal("unlock history lost")
	}
}
