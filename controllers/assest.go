package controllers

import (
	"github.com/astaxie/beego"
	"github.com/hidu/api-front/utils"
	"mime"
	"path/filepath"
)

type AssestController struct {
	beego.Controller
}

func (c *AssestController) Get() {
	fileName := c.Ctx.Input.Param(":splat")
	indexHtml := utils.Assest.GetContent("res/_front_/" + fileName)

	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType != "" {
		c.Ctx.ResponseWriter.Header().Set("Content-Type", mimeType)
	}
	c.Ctx.WriteString(indexHtml)
}
