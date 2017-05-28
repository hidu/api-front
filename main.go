package main

import (
	"github.com/astaxie/beego"
	"github.com/hidu/api-front/models"
	_ "github.com/hidu/api-front/routers"
	"github.com/hidu/api-front/server"
	"log"
)

func main() {
	log.Println("starting...")
	log.SetFlags(log.LstdFlags)
	beego.BConfig.WebConfig.AutoRender = false
	beego.BConfig.ServerName = "api-front"
	beego.BConfig.EnableGzip = true

	models.ORMInit()

	beego.AddAPPStartHook(server.RegisterManager)

	beego.Run()
}
