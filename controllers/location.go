package controllers

import (
	"github.com/hidu/api-front/models"
	//	"fmt"
)

type LocationController struct {
	BaseController
}

func (c *LocationController) Get() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	api := new(models.Location)
	api.ID = id
	err := api.Read()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		c.EchoJson(0, "success", api)
	}
}

func (c *LocationController) Create() {
	location := new(models.Location)
	location.Name = c.Input().Get("name")
	location.Intro = c.Input().Get("intro")
	location.Location = c.Input().Get("location")
	location.NodeId, _ = c.GetInt64("node_id", 0)
	if location.NodeId < 1 {
		c.EchoJson(400, "node_id required", nil)
		return
	}
	location.Status = 1
	id, err := location.Create()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		location.ID = id
		c.EchoJson(0, "success", location)
	}
}

func (c *LocationController) Update() {
	id := c.GetRoutParamInt64(":id")
	if id < 0 {
		c.EchoJson(400, "Bad Request", nil)
		return
	}
	location := new(models.Location)
	location.ID = id
	location.Name = c.Input().Get("name")
	location.Intro = c.Input().Get("intro")
	err := location.Update()
	if err != nil {
		c.EchoJson(500, err.Error(), nil)
	} else {
		location.Read()
		c.EchoJson(0, "success", location)
	}
}
