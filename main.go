package main

import (
	"flag"
	"fmt"
	"github.com/hidu/api-proxy/proxy"
)

var conf_path = flag.String("conf", "./conf/proxy.json", "server conf path")
var conf_demo = flag.Bool("conf_demo", false, "show the demo conf")

func init() {
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\n site: https://github.com/hidu/api-proxy")
		fmt.Println(" version:",proxy.API_PROXY_VERSION)
	}
}

func main() {
	flag.Parse()
	if *conf_demo {
		showConfDemo()
		return
	}
	manager := proxy.NewApiServerManager(*conf_path)
	manager.Start()
}

func showConfDemo() {
	conf := `
{
"server":[
    {"port":8080,"enable":true,"name":"server on 8080 ","note":"8080 note"},
    {"port":8081,"enable":true,"name":"server on 8081 ","note":"8081 note"}
 ]
}
`
	fmt.Println(conf)
}
