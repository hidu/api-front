package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type Api struct {
	Name        string       `json:"-"`
	ConfPath    string       `json:"-"`
	Note        string       `json:"note"`
	TimeoutMs   int          `json:"timeout_ms"`
	Hosts       Hosts        `json:"hosts"`
	Paths       ApiPathMap   `json:"paths"`
	PathsEnable []string     `json:"-"`
	rw          sync.RWMutex `json:"-"`
	Exists      bool         `json:"-"`
}

func NewApi() *Api {
	api := &Api{
		Paths: NewApiPathMap(),
		Hosts: NewHosts(),
	}
	return api
}

func (api *Api) init() {
	log.Println("start load api [", api.Name, "] conf")
	for path_uri, pathObj := range api.Paths {
		pathObj.Path = path_uri
		pathObj.init()
	}
	if api.TimeoutMs < 1 {
		api.TimeoutMs = 5000
	}
	api.Exists = true
}

var pathReg *regexp.Regexp = regexp.MustCompile(`^/[\w-/]*$`)

func (api *Api) IsValidPath(myPath string) bool {
	return pathReg.MatchString(myPath)
}

func (api *Api) UpdateApiPath(name string, apipath *ApiPath) {
	api.rw.Lock()
	defer api.rw.Unlock()
	api.Paths[name] = apipath
}

func (api *Api) deletePath(name string) {
	api.rw.Lock()
	defer api.rw.Unlock()
	if _, has := api.Paths[name]; has {
		delete(api.Paths, name)
	}
	log.Println("deletePath", name)
}

func (api *Api) isPathRegistered(name string) bool {
	_, has := api.Paths[name]
	return has
}

func (api *Api) Save() error {
	data, err := json.MarshalIndent(api, "", "    ")
	if err != nil {
		return err
	}
	oldData, _ := ioutil.ReadFile(api.ConfPath)
	if string(oldData) != string(data) {
		back_path := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(TIME_FORMAT_INT)
		DirCheck(back_path)
		err = ioutil.WriteFile(back_path, oldData, 0644)
		log.Println("backup ", back_path, err)
	}
	err = ioutil.WriteFile(api.ConfPath, data, 0644)
	return err
}

func (api *Api) Delete() {
	back_path := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(TIME_FORMAT_INT)
	DirCheck(back_path)
	err := os.Rename(api.ConfPath, back_path)
	log.Println("backup ", back_path, err)
}

func (api *Api) Clone() *Api {
	data, _ := json.Marshal(api)
	var newApi *Api
	json.Unmarshal(data, &newApi)
	newApi.Name = api.Name
	newApi.ConfPath = api.ConfPath
	newApi.Exists = api.Exists
	newApi.init()
	return newApi
}

func (api *Api) HostRename(orig_name, new_name string) {
	if orig_name == "" || orig_name == new_name {
		return
	}
	if _, has := api.Hosts[orig_name]; has {
		delete(api.Hosts, orig_name)
	}
	api.Paths.HostRename(orig_name, new_name)
}

func (api *Api) HostCheckDelete(host_names []string) {
	tmpMap := make(map[string]int)
	for _, v := range host_names {
		tmpMap[v] = 1
	}

	for n := range api.Hosts {
		if _, has := tmpMap[n]; !has {
			delete(api.Hosts, n)
		}
	}

	api.Paths.HostCheckDelete(tmpMap)
}
