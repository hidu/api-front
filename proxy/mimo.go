package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MimoServer struct {
	Enable     bool
	Apis       map[string]*Api
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
	mimo.Apis = make(map[string]*Api)
	mimo.routers = NewRouters()
	mimo.web = NewWebAdmin(mimo)
	return mimo
}

func (mimo *MimoServer) Start() error {
	addr := fmt.Sprintf(":%d", mimo.ServerConf.Port)

	mimo.loadAllApis()
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

func (mimo *MimoServer) loadAllApis() {
	fileNames, _ := filepath.Glob(mimo.ConfDir + "/*.json")
	for _, fileName := range fileNames {
		apiName := strings.TrimRight(filepath.Base(fileName), ".json")
		mimo.loadApi(apiName)
	}
}
func (mimo *MimoServer) newApi(name string) *Api {
	api := NewApi()
	api.Name = name
	api.ConfPath = fmt.Sprintf("%s/%s.json", mimo.ConfDir, name)
	return api
}

func (mimo *MimoServer) deleteApi(apiName string) {
	mimo.Rw.Lock()
	defer mimo.Rw.Unlock()
	api, has := mimo.Apis[apiName]
	if !has {
		return
	}
	api.Delete()
	delete(mimo.Apis, apiName)
	//@todo
}

func (mimo *MimoServer) loadApi(apiName string) error {
	mimo.Rw.Lock()
	defer mimo.Rw.Unlock()

	api := mimo.newApi(apiName)

	relName, _ := filepath.Rel(filepath.Dir(mimo.ConfDir), api.ConfPath)
	logMsg := fmt.Sprint("load apiule [", relName, "]")

	log.Println(logMsg, "start")

	data, err := ioutil.ReadFile(api.ConfPath)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	err = json.Unmarshal(data, &api)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return err
	}
	log.Println(logMsg, "success")

	api.init()
	api.Exists = true
	mimo.Apis[apiName] = api

	for _, api_path := range api.Paths {
		path_uri := filepath.ToSlash(filepath.Clean(fmt.Sprintf("/%s/%s", apiName, api_path.Path)))
		if strings.HasSuffix(api_path.Path, "/") && !strings.HasSuffix(path_uri, "/") {
			path_uri = path_uri + "/"
		}
		if !api_path.Enable {
			log.Println("apiApi", apiName, path_uri, "no backend,skip")
			continue
		}
		router := NewRouterItem(apiName, path_uri, mimo.newHandler(path_uri, api_path, api))
		mimo.routers.BindRouter(path_uri, router)
	}
	return nil
}

func (mimo *MimoServer) newHandler(path_uri string, api_path *ApiPath, api *Api) func(http.ResponseWriter, *http.Request) {
	log.Println(mimo.ServerConf.Port, api.Name, "bind path [", path_uri, "]")

	return func(rw http.ResponseWriter, req *http.Request) {
		log.Println(req.URL.String())

		relPath := req.URL.Path[len(path_uri):]
		req.Header.Del("Connection")
		body, _ := ioutil.ReadAll(req.Body)
		logData := make(map[string]interface{})
		masterHost := api_path.GetMasterHostName()
		if masterHost == "" {
			masterHost = api.Hosts.GetDefaultHostName()
		}
		defer (func() {
			uri := req.URL.Path
			if req.URL.RawQuery != "" {
				uri += "?" + req.URL.RawQuery
			}
			log.Println(mimo.ServerConf.Port, req.RemoteAddr, req.Method, uri, "master:", masterHost, logData)
		})()

		var wg sync.WaitGroup

		addrInfo := strings.Split(req.RemoteAddr, ":")

		for host_name := range api.Hosts {
			wg.Add(1)
			log.Println("back is", host_name)
			go (func(host_name string, api_path *ApiPath, rw http.ResponseWriter, req *http.Request) {
				defer wg.Done()

				start := time.Now()
				isMaster := masterHost == host_name
				backLog := make(map[string]interface{})
				logData[fmt.Sprintf("back_%s", host_name)] = backLog

				backLog["isMaster"] = isMaster
				host, has := api.Hosts[host_name]
				if !has {
					return
				}
				urlNew := host.Url
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
				if req.Header.Get("Content-Length") != "" {
					reqNew.ContentLength = int64(len(body))
				}
				reqNew.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])

				httpClient := &http.Client{}

				httpClient.Timeout = time.Duration(api.TimeoutMs) * time.Millisecond

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
			})(host_name, api_path, rw, req)
		}
		wg.Wait()

	}
}

func (mimo *MimoServer) getApiByName(name string) *Api {
	if api, has := mimo.Apis[name]; has {
		return api
	}
	return nil
}
