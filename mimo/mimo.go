package mimo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type MimoServer struct {
	Port    int
	Enable  bool
	Modules map[string]*Module
	manager *MimoServerManager
	ConfDir string
	Rw      sync.RWMutex
	routers []*Router
}

type Router struct {
	Path   string
	Hander http.HandlerFunc
}

func NewMimoServer(port int, manager *MimoServerManager) *MimoServer {
	mimo := &MimoServer{Port: port, manager: manager}
	mimo.ConfDir = fmt.Sprintf("%s/module_%d", filepath.Dir(manager.ConfPath), mimo.Port)
	mimo.Modules = make(map[string]*Module)
	mimo.routers = make([]*Router, 0)
	return mimo
}

func (mimo *MimoServer) Start() error {
	addr := fmt.Sprintf(":%d", mimo.Port)

	mimo.loadAllModules()
	log.Println("start server:", addr)
	err := http.ListenAndServe(addr, mimo)
	return err
}

func (mimo *MimoServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, router := range mimo.routers {
		if strings.HasPrefix(req.URL.Path, router.Path) {
			router.Hander(rw, req)
			return
		}
	}
	rw.Write([]byte("hello http mimi transfer"))
}

func (mimo *MimoServer) loadAllModules() {
	fileNames, _ := filepath.Glob(mimo.ConfDir + "/*.json")
	for _, fileName := range fileNames {
		moduleName := strings.TrimRight(filepath.Base(fileName), ".json")
		mimo.loadModule(moduleName)
	}
}

func (mimo *MimoServer) loadModule(moduleName string) error {
	mimo.Rw.Lock()
	defer mimo.Rw.Unlock()

	conf_path := fmt.Sprintf("%s/%s.json", mimo.ConfDir, moduleName)
	relName, _ := filepath.Rel(filepath.Dir(mimo.ConfDir), conf_path)
	logMsg := fmt.Sprint("load module [", relName, "]")

	log.Println(logMsg, "start")

	data, err := ioutil.ReadFile(conf_path)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	var mod *Module
	err = json.Unmarshal(data, &mod)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	log.Println(logMsg, "success")
	mod.Name = moduleName
	mimo.Modules[moduleName] = mod

	for path_name, back := range mod.Paths {
		path_uri := filepath.ToSlash(path.Clean(fmt.Sprintf("/%s/%s", moduleName, path_name)))
		router := &Router{
			Path:   path_uri,
			Hander: mimo.newHandler(path_uri, back),
		}
		mimo.routers = append(mimo.routers, router)
	}

	return nil
}

func (mimo *MimoServer) newHandler(path_uri string, backs Backends) func(http.ResponseWriter, *http.Request) {
	log.Println(mimo.Port, "bind path [", path_uri, "]")
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("hello"))
		rw.Write([]byte(path_uri))
	}
}
