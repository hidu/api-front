package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	//	"path"
	"path/filepath"
	//	"sort"
	"strings"
	"sync"
	"time"
)

type MimoServer struct {
	Enable     bool
	Modules    map[string]*Module
	manager    *MimoServerManager
	ConfDir    string
	Rw         sync.RWMutex
	routers    *Routers
	web        *WebAdmin
	ServerConf *ServerConfItem
}

func NewMimoServer(conf *ServerConfItem, manager *MimoServerManager) *MimoServer {
	mimo := &MimoServer{ServerConf: conf, manager: manager}
	mimo.ConfDir = fmt.Sprintf("%s/api_%d", filepath.Dir(manager.ConfPath), conf.Port)
	mimo.Modules = make(map[string]*Module)
	mimo.routers = NewRouters()
	mimo.web = NewWebAdmin(mimo)
	return mimo
}

func (mimo *MimoServer) Start() error {
	addr := fmt.Sprintf(":%d", mimo.ServerConf.Port)

	mimo.loadAllModules()
	log.Println("start server:", addr)
	err := http.ListenAndServe(addr, mimo)
	return err
}

func (mimo *MimoServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	router := mimo.routers.GetRouterByReqPath(req.URL.Path)
	if router != nil {
		router.Hander.ServeHTTP(rw, req)
		return
	}
	mimo.web.ServeHTTP(rw, req)
}

func (mimo *MimoServer) loadAllModules() {
	fileNames, _ := filepath.Glob(mimo.ConfDir + "/*.json")
	for _, fileName := range fileNames {
		moduleName := strings.TrimRight(filepath.Base(fileName), ".json")
		mimo.loadModule(moduleName)
	}
}
func (mimo *MimoServer) newModule(name string) *Module {
	mod := NewModule()
	mod.Name = name
	mod.ConfPath = fmt.Sprintf("%s/%s.json", mimo.ConfDir, name)
	return mod
}

func (mimo *MimoServer) deleteModule(moduleName string) {
	mimo.Rw.Lock()
	defer mimo.Rw.Unlock()
	mod, has := mimo.Modules[moduleName]
	if !has {
		return
	}
	mod.Delete()
	delete(mimo.Modules, moduleName)
	//@todo
}

func (mimo *MimoServer) loadModule(moduleName string) error {
	mimo.Rw.Lock()
	defer mimo.Rw.Unlock()

	mod := mimo.newModule(moduleName)

	relName, _ := filepath.Rel(filepath.Dir(mimo.ConfDir), mod.ConfPath)
	logMsg := fmt.Sprint("load module [", relName, "]")

	log.Println(logMsg, "start")

	data, err := ioutil.ReadFile(mod.ConfPath)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	err = json.Unmarshal(data, &mod)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	log.Println(logMsg, "success")

	mod.init()
	mod.Exists = true
	mimo.Modules[moduleName] = mod

	for path_name, back := range mod.Paths {
		path_uri := filepath.ToSlash(fmt.Sprintf("/%s/%s", moduleName, strings.TrimLeft(path_name, "/")))

		if len(back) < 1 {
			log.Println("apiModule", moduleName, path_name, "no backend,skip")
			continue
		}
		router := NewRouterItem(moduleName, path_uri, mimo.newHandler(path_uri, back, mod))
		mimo.routers.BindRouter(path_uri, router)
	}
	return nil
}

func (mimo *MimoServer) newHandler(path_uri string, backs Backends, mod *Module) func(http.ResponseWriter, *http.Request) {
	log.Println(mimo.ServerConf.Port, mod.Name, "bind path [", path_uri, "]")

	return func(rw http.ResponseWriter, req *http.Request) {
		log.Println(req.URL.String())

		relPath := req.URL.Path[len(path_uri):]
		req.Header.Del("Connection")
		body, _ := ioutil.ReadAll(req.Body)
		masterIndex := backs.GetMasterIndex()
		logData := make(map[string]interface{})

		defer (func() {
			uri := req.URL.Path
			if req.URL.RawQuery != "" {
				uri += "?" + req.URL.RawQuery
			}
			log.Println(mimo.ServerConf.Port, req.RemoteAddr, req.Method, uri, "master:", masterIndex, logData)
		})()

		var wg sync.WaitGroup

		addrInfo := strings.Split(req.RemoteAddr, ":")

		for n, back := range backs {
			wg.Add(1)
			log.Println("back is", back.Url)
			go (func(index int, back *Backend, rw http.ResponseWriter, req *http.Request) {
				defer wg.Done()

				start := time.Now()
				isMaster := masterIndex == index
				backLog := make(map[string]interface{})
				logData[fmt.Sprintf("back_%d", index)] = backLog

				backLog["isMaster"] = isMaster

				urlNew := back.Url
				if strings.HasSuffix(urlNew, "/") {
					urlNew += strings.TrimLeft(relPath, "/")
				} else {
					urlNew += relPath
				}
				if req.URL.RawQuery != "" {
					urlNew += "?" + req.URL.RawQuery
				}
				backLog["url"] = urlNew

				reqNew, _ := http.NewRequest(req.Method, urlNew, ioutil.NopCloser(bytes.NewReader(body)))
				reqNew.Header = req.Header
				reqNew.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])

				httpClient := &http.Client{}

				httpClient.Timeout = time.Duration(mod.TimeoutMs) * time.Millisecond

				resp, err := httpClient.Do(reqNew)

				rw.Header().Set("Server", "api-proxy")

				if err != nil {
					log.Println("fetch "+urlNew, err)
					if isMaster {
						rw.WriteHeader(http.StatusBadGateway)
						rw.Header().Set("back-url", urlNew)
						rw.Write([]byte("mimo error:" + err.Error()))
					}
					return
				}
				defer resp.Body.Close()
				if isMaster {
					for k, vs := range resp.Header {
						for _, v := range vs {
							rw.Header().Add(k, v)
						}
					}
					rw.Header().Set("back-url", urlNew)
					n, err := io.Copy(rw, resp.Body)
					if err != nil {
						log.Println(urlNew, "io.copy:", n, err)
					}
				}

				used := time.Now().Sub(start)
				backLog["used"] = fmt.Sprintf("%.3f ms", float64(used.Nanoseconds())/1e6)
			})(n, back, rw, req)
		}
		wg.Wait()

	}
}

func (mimo *MimoServer) getModuleByName(name string) *Module {
	if mod, has := mimo.Modules[name]; has {
		return mod
	}
	return nil
}
