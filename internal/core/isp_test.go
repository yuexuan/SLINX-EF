package core

import (
 "encoding/json"
 "os"
 "path/filepath"
 "testing"
 "github.com/slinxlink/node/internal/database"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
)

func TestISPConfigFailClosed(t *testing.T) {
 var err error
 database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
 if err != nil { t.Fatal(err) }
 database.DB.AutoMigrate(&database.Core{}, &database.Config{}, &database.Endpoint{}, &database.Inbound{}, &database.User{}, &database.Rule{})
 Default.ConfigPath=filepath.Join(t.TempDir(),"config.json")
 for _, typ := range []string{"socks","http"} {
  ep:=database.Endpoint{Enable:true,Tag:"isp",Type:typ,Server:"127.0.0.1",ServerPort:1080,Username:"secret-user",Password:"secret-password"}
  database.DB.Create(&ep)
  database.DB.Create(&database.Rule{Sort:1,Key:"inbound",Value:`["443","8443"]`})
  database.DB.Create(&database.Rule{Sort:1,Key:"outbound",Value:`"isp"`})
  for _, state := range []string{"enabled","disabled","deleted"} {
   if state=="disabled" { database.DB.Model(&ep).Update("enable",false) }
   if state=="deleted" { database.DB.Delete(&ep) }
   if err:=generateConfig(); err!=nil {t.Fatal(err)}
   data,_:=os.ReadFile(Default.ConfigPath)
   var cfg config; if err:=json.Unmarshal(data,&cfg); err!=nil {t.Fatal(err)}
   target:="isp"; if state!="enabled" {target="block"}
   if cfg.Route.Rules[1]["outbound"]!=target {t.Fatalf("%s %s leaked route: %s",typ,state,data)}
   if state=="enabled" { ob:=cfg.Outbounds[2]; if ob.Type!=typ || ob.Password!="secret-password" || ob.ServerPort!=1080 {t.Fatalf("bad proxy: %+v",ob)} }
  }
  database.DB.Where("1=1").Delete(&database.Rule{})
 }
}
