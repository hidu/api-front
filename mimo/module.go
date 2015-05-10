package mimo

import (
	"log"
)

type Module struct {
	Name      string
	Desc      string              `json:"desc"`
	TimeoutMs int                 `json:"timeout_ms'`
	Enable    bool                `json:"enable"`
	Paths     map[string]Backends `json:"paths"`
}

func (module *Module) init() {
	log.Println("start load module [", module.Name, "] conf")
	for _, backs := range module.Paths {
		backs.init()
	}
	if module.TimeoutMs < 1 {
		module.TimeoutMs = 5000
	}
}
