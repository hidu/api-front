package main

import (
	"flag"
	"fmt"
	"github.com/hidu/api-front/proxy"
	"log"
	"strings"
)

var confPath = flag.String("conf", "./conf/server.json", "server conf path")
var confDemo = flag.Bool("conf_demo", false, "show the demo conf")

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Ldate)
	ua := flag.Usage
	flag.Usage = func() {
		ua()
		fmt.Println("\n site: https://github.com/hidu/api-front")
		fmt.Println(" version:", proxy.APIFrontVersion)
	}
}

func main() {
	flag.Parse()
	if *confDemo {
		showConfDemo()
		return
	}
	manager := proxy.NewAPIServerManager(*confPath)
	manager.Start()
}

func showConfDemo() {
	conf := `
1. conf/server.json:
---------------------------------------------------------
{
 "users":["admin"]
}
---------------------------------------------------------

2. conf/users:
---------------------------------------------------------
id:admin psw_md5:7bb483729b5a8e26f73e1831cde5b842 psw:psw
---------------------------------------------------------

3. conf/vhost/8080.json
---------------------------------------------------------
{
    "port": 8080,
    "group": "order",
    "name": "order api",
    "enable": true,
    "hidden_cookie": true,
    "note": "订单相关接口",
    "users": [
        "test"
    ]
}
---------------------------------------------------------
`
	fmt.Println(strings.TrimSpace(conf))
}
