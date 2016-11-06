package controllers

import (
	"github.com/astaxie/beego"
	"github.com/hidu/api-front/utils"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	indexHtml := utils.Assest.GetContent("res/index.html")
	c.Ctx.WriteString(indexHtml)
}
