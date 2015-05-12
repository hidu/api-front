package main

import (
	"flag"
	"github.com/hidu/api-proxy/proxy"
)

var conf_path = flag.String("conf", "./conf/proxy.json", "server conf path")

func main() {
	flag.Parse()
	manager := proxy.NewMimoServerManager(*conf_path)
	manager.Start()
}
