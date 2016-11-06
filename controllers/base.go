package controllers

import (
	"github.com/astaxie/beego"
	//	"fmt"
	"strconv"
)

type BaseController struct {
	beego.Controller
}

func (this *BaseController) EchoJson(errno int, errmsg string, data interface{}) {
	result := make(map[string]interface{})
	result["errno"] = errno
	result["errmsg"] = errmsg
	result["data"] = data
	this.Data["json"] = result
	this.ServeJSON()
}

func (this *BaseController) GetRoutParamInt64(name string) int64 {
	val := this.Ctx.Input.Param(name)
	intV, _ := strconv.ParseInt(val, 10, 64)
	return intV
}
