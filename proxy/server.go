package proxy

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ApiServer struct {
	Enable     bool
	Apis       map[string]*Api
	manager    *ApiServerManager
	ConfDir    string
	Rw         sync.RWMutex
	routers    *Routers
	web        *WebAdmin
	ServerConf *ServerConfItem
	counter    *Counter
}

func NewApiServer(conf *ServerConfItem, manager *ApiServerManager) *ApiServer {
	apiServer := &ApiServer{ServerConf: conf, manager: manager}
	apiServer.ConfDir = fmt.Sprintf("%s/api_%d", filepath.Dir(manager.ConfPath), conf.Port)
	apiServer.Apis = make(map[string]*Api)
	apiServer.routers = newRouters()
	apiServer.web = NewWebAdmin(apiServer)
	apiServer.counter = newCounter(apiServer)
	return apiServer
}

func (apiServer *ApiServer) Start() error {
	addr := fmt.Sprintf(":%d", apiServer.ServerConf.Port)

	apiServer.loadAllApis()
	log.Println("start server:", addr)
	err := http.ListenAndServe(addr, apiServer)
	return err
}

func (apiServer *ApiServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	router := apiServer.routers.getRouterByReqPath(req.URL.Path)
	if router != nil {
		router.Hander.ServeHTTP(rw, req)
		return
	}

	if strings.HasPrefix(req.URL.Path, "/_") || req.URL.Path == "/" {
		apiServer.web.ServeHTTP(rw, req)
	} else {
		http.Error(rw, "Api Not Found (api-proxy)", http.StatusNotFound)
	}
}

func (apiServer *ApiServer) loadAllApis() {
	fileNames, _ := filepath.Glob(apiServer.ConfDir + "/*.json")
	for _, fileName := range fileNames {
		log.Println("start load conf file:", fileName)

		baseName := filepath.Base(fileName)

		apiName := baseName[:len(baseName)-5]

		if strings.HasPrefix(apiName, "_") {
			continue
		}

		apiServer.loadAPI(apiName)
	}
}

func (apiServer *ApiServer) deleteAPI(apiName string) {
	apiServer.Rw.Lock()
	defer apiServer.Rw.Unlock()
	api, has := apiServer.Apis[apiName]
	if !has {
		return
	}
	api.delete()
	delete(apiServer.Apis, apiName)
}

func (apiServer *ApiServer) newAPI(apiName string) *Api {
	return newAPI(apiServer, apiName)
}

func (apiServer *ApiServer) getConfDir() string {
	return apiServer.ConfDir
}

func (apiServer *ApiServer) loadAPI(apiName string) error {
	apiServer.Rw.Lock()
	defer apiServer.Rw.Unlock()

	api, err := loadAPIByConf(apiServer, apiName)
	if err != nil {
		log.Println("load api failed,", apiName, err)
		return err
	}

	log.Printf("load api [%s] success", apiName)

	apiServer.Apis[apiName] = api
	if api.Enable {
		router := newRouterItem(apiName, api.Path, apiServer.newHandler(api))
		apiServer.routers.bindRouter(api.Path, router)
	} else {
		apiServer.routers.deleteRouterByPath(api.Path)
		log.Printf("api [%s] is not enable,skip", apiName)
	}

	return nil
}

func (apiServer *ApiServer) uniqReqID(id uint64) string {
	return fmt.Sprintf("%s_%d", time.Now().Format("20060102_150405"), id)
}

func (apiServer *ApiServer) getAPIByName(name string) *Api {
	if api, has := apiServer.Apis[name]; has {
		return api
	}
	return nil
}

func (apiServer *ApiServer) getAPIByPath(bindPath string) *Api {
	bindPath = URLPathClean(bindPath)
	for _, api := range apiServer.Apis {
		if api.Path == bindPath {
			return api
		}
	}
	return nil
}

func (apiServer *ApiServer) GetCounter() *Counter {
	return apiServer.counter
}
