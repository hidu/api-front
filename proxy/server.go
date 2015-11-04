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

// APIServer api server
type APIServer struct {
	Enable  bool
	Apis    map[string]*apiStruct
	manager *APIServerManager
	//current apiServer's conf dir
	ConfDir    string
	Rw         sync.RWMutex
	routers    *routers
	web        *webAdmin
	ServerConf *serverConfItem
	counter    *Counter
}

func newAPIServer(conf *serverConfItem, manager *APIServerManager) *APIServer {
	apiServer := &APIServer{ServerConf: conf, manager: manager}
	apiServer.ConfDir = fmt.Sprintf("%sapi_%d", manager.rootConfDir(), conf.Port)
	if conf.SubDoamin != "" {
		apiServer.ConfDir += "_" + conf.SubDoamin
	}
	apiServer.ConfDir += string(filepath.Separator)

	apiServer.Apis = make(map[string]*apiStruct)
	apiServer.routers = newRouters()
	apiServer.web = newWebAdmin(apiServer)
	apiServer.counter = newCounter(apiServer)
	apiServer.loadAllApis()
	return apiServer
}

func (apiServer *APIServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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

func (apiServer *APIServer) loadAllApis() {
	fileNames, _ := filepath.Glob(apiServer.ConfDir + string(filepath.Separator) + "*.json")
	for _, fileName := range fileNames {
		log.Println("start load conf file:", fileName)

		_, baseName := filepath.Split(fileName)

		if baseName == "" {
			log.Println("skip api conf:", fileName)
			continue
		}

		apiName := baseName[:len(baseName)-5]

		if strings.HasPrefix(apiName, "_") {
			log.Println("skip api", apiName)
			continue
		}

		apiServer.loadAPI(apiName)
	}
}

//func (apiServer *APIServer) deleteAPI(apiName string) {
//	apiServer.Rw.Lock()
//	defer apiServer.Rw.Unlock()
//	api, has := apiServer.Apis[apiName]
//	if !has {
//		return
//	}
//	api.delete()
//	delete(apiServer.Apis, apiName)
//}

func (apiServer *APIServer) unRegisterAPI(apiName string) {
	apiServer.Rw.Lock()
	defer apiServer.Rw.Unlock()
	_, has := apiServer.Apis[apiName]
	if !has {
		return
	}
	delete(apiServer.Apis, apiName)
}

func (apiServer *APIServer) newAPI(apiName string) *apiStruct {
	return newAPI(apiServer, apiName)
}

func (apiServer *APIServer) getConfDir() string {
	return apiServer.ConfDir
}

func (apiServer *APIServer) rootConfDir() string {
	return apiServer.manager.rootConfDir()
}

func (apiServer *APIServer) loadAPI(apiName string) error {
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

func (apiServer *APIServer) uniqReqID(id uint64) string {
	return fmt.Sprintf("%s_%d", time.Now().Format("20060102_150405"), id)
}

func (apiServer *APIServer) serverName() string {
	return fmt.Sprintf("%s:%d", apiServer.ServerConf.SubDoamin, apiServer.ServerConf.Port)
}
func (apiServer *APIServer) subDomain() string {
	return apiServer.ServerConf.SubDoamin
}

func (apiServer *APIServer) getAPIByName(name string) *apiStruct {
	if name == "" {
		return nil
	}
	if api, has := apiServer.Apis[name]; has {
		return api
	}
	return nil
}

func (apiServer *APIServer) getAPIByPath(bindPath string) *apiStruct {
	bindPath = URLPathClean(bindPath)
	for _, api := range apiServer.Apis {
		if api.Path == bindPath {
			return api
		}
	}
	return nil
}

// GetCounter get counter
func (apiServer *APIServer) GetCounter() *Counter {
	return apiServer.counter
}

func (apiServer *APIServer) hasUser(name string) bool {
	if apiServer.ServerConf.users != nil && apiServer.ServerConf.users.hasUser(name) {
		return true
	}
	if apiServer.manager.serverConf.Users != nil && apiServer.manager.serverConf.Users.hasUser(name) {
		return true
	}
	return false
}
