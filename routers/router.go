package routers

import (
	"github.com/astaxie/beego"
	"github.com/hidu/api-front/controllers"
)

func init() {
	beego.Router("/", &controllers.MainController{})

	beego.Router("/group/create", &controllers.GroupController{}, "get:Create")
	beego.Router(`/group/get/:id(\d+)`, &controllers.GroupController{}, "get:Get")
	beego.Router(`/group/update/:id(\d+)`, &controllers.GroupController{}, "post:Update")

	beego.Router("/host/create", &controllers.HostController{}, "get:Create")
	beego.Router(`/host/get/:id(\d+)`, &controllers.HostController{}, "get:Get")
	beego.Router(`/host/update/:id(\d+)`, &controllers.HostController{}, "post:Update")

	beego.Router("/location/create", &controllers.LocationController{}, "get:Create")
	beego.Router(`/location/get/:id(\d+)`, &controllers.LocationController{}, "get:Get")
	beego.Router(`/location/update/:id(\d+)`, &controllers.LocationController{}, "post:Update")

	beego.Router("/_front_/*", &controllers.AssestController{})
}
