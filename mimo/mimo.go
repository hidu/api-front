package mimo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MimoServer struct {
	Port    int
	Enable  bool
	Modules map[string]*Module
	manager *MimoServerManager
	ConfDir string
	Rw      sync.RWMutex
	routers Routers
}
type Routers []*Router

type Router struct {
	Path   string
	Hander http.HandlerFunc
}

func (rs Routers) Len() int {
	return len(rs)
}

func (rs Routers) Less(i, j int) bool {
	return len(rs[i].Path) > len(rs[j].Path)
}

func (rs Routers) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
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
	mimo.Rw.RLock()
	defer mimo.Rw.RUnlock()
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
	mod.init()
	mimo.Modules[moduleName] = mod

	for path_name, back := range mod.Paths {
		path_uri := filepath.ToSlash(path.Clean(fmt.Sprintf("/%s/%s", moduleName, path_name)))
		router := &Router{
			Path:   path_uri,
			Hander: mimo.newHandler(path_uri, back, mod),
		}
		mimo.routers = append(mimo.routers, router)
	}
	sort.Sort(mimo.routers)
	return nil
}

func (mimo *MimoServer) newHandler(path_uri string, backs Backends, mod *Module) func(http.ResponseWriter, *http.Request) {
	log.Println(mimo.Port, "bind path [", path_uri, "]")
	return func(rw http.ResponseWriter, req *http.Request) {
		relPath := req.URL.Path[len(path_uri):]
		req.Header.Del("Connection")
		body, _ := ioutil.ReadAll(req.Body)
		masterIndex := backs.GetMasterIndex()
		logData := make(map[string]interface{})

		defer (func() {
			log.Println(mimo.Port, req.RemoteAddr, req.Method, req.URL.Path, "master:", masterIndex, logData)
		})()

		var wg sync.WaitGroup

		for n, back := range backs {
			wg.Add(1)
			go (func(index int, back *Backend, rw http.ResponseWriter, req *http.Request) {
				defer wg.Done()

				start := time.Now()
				isMaster := masterIndex == n
				backLog := make(map[string]interface{})
				logData[fmt.Sprintf("back_%d", n)] = backLog

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

				httpClient := &http.Client{}

				httpClient.Timeout = time.Duration(mod.TimeoutMs) * time.Microsecond

				resp, err := httpClient.Do(reqNew)

				if err != nil {
					log.Println("Error Fetching "+urlNew, err)
					rw.WriteHeader(http.StatusBadGateway)
					rw.Write([]byte("mimo error:" + err.Error()))
					return
				}
				defer resp.Body.Close()
				if isMaster {
					for k, vs := range resp.Header {
						for _, v := range vs {
							rw.Header().Add(k, v)
						}
					}
					n, err := io.Copy(rw, resp.Body)
					log.Println("io.copy:", n, err)
				}

				used := time.Now().Sub(start)
				backLog["used"] = fmt.Sprintf("%.3f ms", float64(used.Nanoseconds())/1e6)
			})(n, back, rw, req)
		}
		wg.Wait()

	}
}
