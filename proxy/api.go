package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

//Api   一般是一个模块
type Api struct {
	Name        string       `json:"-"`
	ConfPath    string       `json:"-"`
	Path        string       `json:"path"`
	Note        string       `json:"note"`
	TimeoutMs   int          `json:"timeout_ms"`
	Hosts       Hosts        `json:"hosts"`
	Enable      bool         `json:"enable"`
	Caller      Caller       `json:"caller"`
	rw          sync.RWMutex `json:"-"`
	Exists      bool         `json:"-"`
	HostAsProxy bool         `json:"host_as_proxy"` //是否把后端当作代理
	Pv          uint64       `json:"-"`
	LastVisit   time.Time    `json:"-"`       //最后访问时间
	Version     int64        `json:"version"` //配置文件的版本号
	apiServer   *ApiServer
}

// init new api for server
func NewApi(apiServer *ApiServer, apiName string) *Api {
	api := &Api{
		Name:      apiName,
		ConfPath:  fmt.Sprintf("%s/%s.json", apiServer.GetConfDir(), apiName),
		Hosts:     NewHosts(),
		apiServer: apiServer,
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
		item, _ := NewCallerItem(IP_ALL)
		item.Enable = true
		item.Note = "default all"
		api.Caller.AddNewCallerItem(item)
	}
	if api.Path != "" {
		api.Path = UrlPathClean(api.Path)
	}
	api.Caller.Sort()
	err = api.Caller.Init()

	api.Exists = true
	return err
}

var pathReg= regexp.MustCompile(`^/[\w-/]+/$`)

var ApiNameReg= regexp.MustCompile(`^[\w-]+$`)

func (api *Api) IsValidPath(myPath string) bool {
	return pathReg.MatchString(myPath)
}

func (api *Api) Save() error {
	api.rw.Lock()
	defer api.rw.Unlock()

	data, err := json.MarshalIndent(api, "", "    ")
	if err != nil {
		return err
	}
	oldData, _ := ioutil.ReadFile(api.ConfPath)
	if string(oldData) != string(data) {
		backPath := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(TIME_FORMAT_INT)
		DirCheck(backPath)
		err = ioutil.WriteFile(backPath, oldData, 0644)
		log.Println("backup ", backPath, err)
	}
	err = ioutil.WriteFile(api.ConfPath, data, 0644)
	return err
}

func (api *Api) Delete() {
	api.rw.Lock()
	defer api.rw.Unlock()
	backPath := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(TIME_FORMAT_INT)
	DirCheck(backPath)
	err := os.Rename(api.ConfPath, backPath)
	log.Println("backup ", backPath, err)
}

func (api *Api) Clone() *Api {
	api.rw.RLock()
	defer api.rw.RUnlock()
	data, _ := json.Marshal(api)
	var newApi *Api
	json.Unmarshal(data, &newApi)
	newApi.Name = api.Name
	newApi.ConfPath = api.ConfPath
	newApi.Exists = api.Exists
	newApi.init()
	newApi.apiServer = api.apiServer
	newApi.Hosts.Init()
	return newApi
}

func (api *Api) HostRename(origName, newName string) {
	if origName == "" || origName == newName {
		return
	}
	api.rw.Lock()
	defer api.rw.Unlock()

	if _, has := api.Hosts[origName]; has {
		delete(api.Hosts, origName)
	}
}

func (api *Api) HostCheckDelete(hostNames []string) {

	api.rw.Lock()
	defer api.rw.Unlock()

	tmpMap := make(map[string]int)
	for _, v := range hostNames {
		tmpMap[v] = 1
	}

	for n := range api.Hosts {
		if _, has := tmpMap[n]; !has {
			delete(api.Hosts, n)
		}
	}

}

func (api *Api) GetMasterHostName(cpf *CallerPrefConf) string {
	api.rw.RLock()
	defer api.rw.RUnlock()

	names := make([]string, 0)
	for name, host := range api.Hosts {
		if host.Enable {
			names = append(names, name)
		}
	}
	return api.Caller.GetPrefHostName(names, cpf)
}

func (api *Api) CookieName() string {
	return ApiCookieName(api.Name)
}

func LoadApiByConf(apiServer *ApiServer, apiName string) (*Api, error) {
	api := NewApi(apiServer, apiName)
	relName, _ := filepath.Rel(filepath.Dir(apiServer.GetConfDir()), api.ConfPath)
	logMsg := fmt.Sprint("load api [", apiName, "],[", relName, "]")

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

func (api *Api) PvInc() uint64 {
	return api.apiServer.GetCounter().PvInc(api.Name)
}

func (api *Api) GetPv() uint64 {
	return api.apiServer.GetCounter().GetPv(api.Name)
}

func (api *Api) GetRoomName() string {
	return fmt.Sprintf("_room_%s", api.Name)
}

func ApiCookieName(apiName string) string {
	return fmt.Sprintf("%s_%s", API_PREF_PARMA_NAME, apiName)
}

/**
* get sorted hosts,master is at first
 */
func (api *Api) getApiHostsByReq(req *http.Request) (hs []*Host, master string, cpf *CallerPrefConf) {
	cpf = NewCallerPrefConfByHttpRequest(req, api)
	caller := api.Caller.getCallerItemByIp(cpf.GetIp())
	masterHost := api.GetMasterHostName(cpf)

	hs = make([]*Host, 0)
	hsTmp := make([]*Host, 0)
	for _, apiHost := range api.Hosts {
		if !apiHost.Enable || caller.IsHostIgnore(apiHost.Name) {
			continue
		}
		if apiHost.Name == masterHost {
			hs = append(hs, apiHost)
		} else {
			hsTmp = append(hsTmp, apiHost)
		}
	}
	hs = append(hs, hsTmp...)
	return hs, masterHost, cpf
}
