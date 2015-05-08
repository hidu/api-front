package main

import (
	"flag"
	"github.com/hidu/mimo/mimo"
)

var conf_path = flag.String("conf", "./conf/mimo.json", "server conf path")

func main() {
	flag.Parse()
	manager := mimo.NewMimoServerManager(*conf_path)
	manager.Start()
}
