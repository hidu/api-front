package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"regexp"
	"sync"
)

type Module struct {
	Name      string              `json:"-"`
	ConfPath  string              `json:"-"`
	Note      string              `json:"note"`
	TimeoutMs int                 `json:"timeout_ms"`
	Paths     map[string]Backends `json:"paths"`
	rw        sync.RWMutex        `json:"-"`
	Exists    bool                `json:"-"`
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
	module.Exists = true
}

var pathReg *regexp.Regexp = regexp.MustCompile(`^/[\w-/]*$`)

func (module *Module) IsValidPath(myPath string) bool {
	return pathReg.MatchString(myPath)
}

func (module *Module) UpdateBackends(name string, backs Backends) {
	module.rw.Lock()
	defer module.rw.Unlock()
	module.Paths[name] = backs
	log.Println("UpdateBackends:", name, backs)
}

func (module *Module) deletePath(name string) {
	module.rw.Lock()
	defer module.rw.Unlock()
	if _, has := module.Paths[name]; has {
		delete(module.Paths, name)
	}
	log.Println("deletePath", name)
}

func (module *Module) isPathRegistered(name string) bool {
	_, has := module.Paths[name]
	return has
}

func (module *Module) Save() error {
	data, err := json.MarshalIndent(module, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(module.ConfPath, data, 0655)
}

func (module *Module) Clone() *Module {
	data, _ := json.Marshal(module)
	var mod *Module
	json.Unmarshal(data, &mod)
	mod.Name = module.Name
	mod.ConfPath = module.ConfPath
	return mod
}
