package main

import (
	"embed"
	"os"
 "fmt"

	"github.com/slinxlink/node/internal/app"
	"github.com/slinxlink/node/internal/bootstrap"
	"github.com/slinxlink/node/internal/cli"
)

var Version = "dev"
var Commit = "unknown"

//go:embed web/dist
var fs embed.FS

func main() {
 if len(os.Args)>1 && os.Args[1]=="version" { fmt.Printf("SLINX-EF %s %s\n",Version,Commit); return }
	os.Chdir("var")

	if len(os.Args) > 1 && os.Args[1] == "cli" {
		cli.Start(Version)
		return
	}

	app.Version = Version
	bootstrap.Boot(fs)
}
