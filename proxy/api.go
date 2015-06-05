package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type Api struct {
	Name      string       `json:"-"`
	ConfPath  string       `json:"-"`
	Path      string       `json:"path"`
	Note      string       `json:"note"`
	TimeoutMs int          `json:"timeout_ms"`
	Hosts     Hosts        `json:"hosts"`
	Enable    bool         `json:"enable"`
	Caller    Caller       `json:"caller"`
	rw        sync.RWMutex `json:"-"`
	Exists    bool         `json:"-"`
}

func NewApi(confDir string, apiName string) *Api {
	api := &Api{
		Name:     apiName,
		ConfPath: fmt.Sprintf("%s/%s.json", confDir, apiName),
		Hosts:    NewHosts(),
	}
	return api
}

func (api *Api) init() (err error) {
	log.Println("start load api [", api.Name, "] conf")

	if api.TimeoutMs < 1 {
		api.TimeoutMs = 5000
	}
	if api.Caller == nil {
		api.Caller = NewCaller()
		item, _ := NewCallerItem("*.*.*.*")
		item.Enable = true
		item.Note = "all"
		api.Caller.AddNewCallerItem(item)
	}
	api.Caller.Sort()
	err = api.Caller.Init()

	api.Exists = true
	return err
}

var pathReg *regexp.Regexp = regexp.MustCompile(`^/[\w-/]+/$`)

func (api *Api) IsValidPath(myPath string) bool {
	return pathReg.MatchString(myPath)
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

}

func (api *Api) GetMasterHostName(cpf *CallerPrefConf) string {
	names := make([]string, 0, len(api.Hosts))
	for name := range api.Hosts {
		names = append(names, name)
	}
	return api.Caller.GetPrefHostName(names, cpf)
}

func LoadApiByConf(confDir string, apiName string) (*Api, error) {
	api := NewApi(confDir, apiName)
	relName, _ := filepath.Rel(filepath.Dir(confDir), api.ConfPath)
	logMsg := fmt.Sprint("load api module [", relName, "]")

	log.Println(logMsg, "start")

	data, err := ioutil.ReadFile(api.ConfPath)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return api, err
	}
	err = json.Unmarshal(data, &api)
	if err != nil {
		log.Println(logMsg, "failed,", err)
		return api, err
	}
	api.Hosts.Init()
	log.Println(logMsg, "success")
	if api.Path == "" {
		api.Path = fmt.Sprintf("/%s/", apiName)
	}
	if !api.IsValidPath(api.Path) {
		return api, fmt.Errorf("path wrong:", api.Path)
	}

	err = api.init()
	api.Exists = true
	return api, err
}
