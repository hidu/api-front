package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
)

type Module struct {
	Name      string              `json:"-"`
	ConfPath  string              `json:"-"`
	Note      string              `json:"note"`
	TimeoutMs int                 `json:"timeout_ms"`
	Paths     map[string]Backends `json:"paths"`
	rw        sync.RWMutex        `json:"-"`
}

func NewModule() *Module {
	mod := &Module{
		Paths: make(map[string]Backends),
	}
	return mod
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

func (module *Module) UpdateBackends(name string, backs Backends) {
	module.rw.Lock()
	defer module.rw.Unlock()
	module.Paths[name] = backs
}

func (module *Module) Save() error {
	data, err := json.MarshalIndent(module, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(module.ConfPath, data, 0655)
}
