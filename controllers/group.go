package controllers

import (
	"github.com/hidu/api-front/models"
	//	"fmt"
)

type GroupController struct {
	BaseController
}

func (c *GroupController) Get() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	group := new(models.ServerGroup)
	group.ID = id
	err := group.Read()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		c.EchoJson(0, "success", group)
	}
}

func (c *GroupController) Create() {
	group := new(models.ServerGroup)
	group.Name = c.Input().Get("name")
	group.Intro = c.Input().Get("intro")
	group.HomePage = c.Input().Get("home_page")
	group.Status = 1
	id, err := group.Create()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		group.ID = id
		c.EchoJson(0, "success", group)
	}
}

func (c *GroupController) Update() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	group := new(models.ServerGroup)
	group.ID = id
	group.Name = c.Input().Get("name")
	group.Intro = c.Input().Get("intro")
	group.HomePage = c.Input().Get("home_page")
	err := group.Update()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		group.Read()
		c.EchoJson(0, "success", group)
	}
}
