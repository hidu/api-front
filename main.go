package main

import (
	"flag"
	"fmt"
	"github.com/hidu/api-man/proxy"
	"log"
)

var confPath = flag.String("conf", "./conf/man.json", "server conf path")
var confDemo = flag.Bool("conf_demo", false, "show the demo conf")

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Ldate)
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\n site: https://github.com/hidu/api-man")
		fmt.Println(" version:", proxy.API_PROXY_VERSION)
	}
}

func main() {
	flag.Parse()
	if *confDemo {
		showConfDemo()
		return
	}
	manager := proxy.NewApiServerManager(*confPath)
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
