package main

import (
	_ "github.com/hidu/api-front/routers"
	"github.com/astaxie/beego"
)

func main() {
	
	beego.BConfig.WebConfig.AutoRender=false;
	beego.Run()
}
