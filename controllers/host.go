package controllers

import (
	"github.com/hidu/api-front/models"
	//	"fmt"
)

type HostController struct {
	BaseController
}

func (c *HostController) Get() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	group := new(models.ServerHost)
	group.ID = id
	err := group.Read()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		c.EchoJson(0, "success", group)
	}
}

func (c *HostController) Create() {
	server := new(models.ServerHost)
	var err error
	server.Name = c.Input().Get("name")
	server.Intro = c.Input().Get("intro")
	server.Port, err = c.GetInt("port")
	server.UniqKey = c.GetString("uniq_key")
	server.GroupID, err = c.GetInt64("group_id")

	server.Status = 1
	id, err := server.Create()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		server.ID = id
		c.EchoJson(0, "success", server)
	}
}

func (c *HostController) Update() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	server := new(models.ServerHost)
	server.ID = id
	server.Name = c.Input().Get("name")
	server.Intro = c.Input().Get("intro")
	err := server.Update()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		server.Read()
		c.EchoJson(0, "success", server)
	}
}
