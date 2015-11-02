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

type apiStruct struct {
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
	apiServer   *APIServer
}

// init new api for server
func newAPI(apiServer *APIServer, apiName string) *apiStruct {
	api := &apiStruct{
		Name:      apiName,
		Hosts:     newHosts(),
		apiServer: apiServer,
	}
	api.ConfPath = api.getConfPath()
	return api
}

func (api *apiStruct) getConfPath() string {
	return fmt.Sprintf("%s/%s.json", api.apiServer.getConfDir(), api.Name)
}

func (api *apiStruct) init() (err error) {
	log.Println("start load api [", api.Name, "] conf")

	if api.TimeoutMs < 1 {
		api.TimeoutMs = 5000
	}
	if api.Caller == nil {
		api.Caller = newCaller()
		item, _ := newCallerItem(ipAll)
		item.Enable = true
		item.Note = "default all"
		api.Caller.addNewCallerItem(item)
	}
	if api.Path != "" {
		api.Path = URLPathClean(api.Path)
	}
	api.Caller.Sort()
	err = api.Caller.init()

	api.Exists = true
	return err
}

var pathReg = regexp.MustCompile(`^/([\w-/]+/)*$`)

var apiNameReg = regexp.MustCompile(`^[\w-]+$`)

func (api *apiStruct) isValidPath(myPath string) bool {
	return pathReg.MatchString(myPath)
}

func (api *apiStruct) save() error {
	api.rw.Lock()
	defer api.rw.Unlock()

	data, err := json.MarshalIndent(api, "", "    ")
	if err != nil {
		return err
	}
	oldData, _ := ioutil.ReadFile(api.ConfPath)
	if string(oldData) != string(data) {
		backPath := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(timeFormatInt)
		DirCheck(backPath)
		err = ioutil.WriteFile(backPath, oldData, 0644)
		log.Println("backup ", backPath, err)
	}
	err = ioutil.WriteFile(api.ConfPath, data, 0644)
	return err
}

func (api *apiStruct) delete() error {
	api.rw.Lock()
	defer api.rw.Unlock()
	backPath := filepath.Dir(api.ConfPath) + "/_back/" + filepath.Base(api.ConfPath) + "." + time.Now().Format(timeFormatInt)
	DirCheck(backPath)
	err := os.Rename(api.ConfPath, backPath)
	log.Println("backup ", backPath, err)
	return err
}

func (api *apiStruct) reName(newName string) error {
	if api.Name == newName {
		log.Println("rename skip,not change,newName:", newName)
		return nil
	}
	err := api.delete()
	if err != nil {
		return err
	}
	api.Name = newName
	api.ConfPath = api.getConfPath()
	return api.save()
}

func (api *apiStruct) clone() *apiStruct {
	api.rw.RLock()
	defer api.rw.RUnlock()
	data, _ := json.Marshal(api)
	var newAPI *apiStruct
	json.Unmarshal(data, &newAPI)
	newAPI.Name = api.Name
	newAPI.ConfPath = api.ConfPath
	newAPI.Exists = api.Exists
	newAPI.init()
	newAPI.apiServer = api.apiServer
	newAPI.Hosts.init()
	return newAPI
}

func (api *apiStruct) hostRename(origName, newName string) {
	if origName == "" || origName == newName {
		return
	}
	api.rw.Lock()
	defer api.rw.Unlock()

	if _, has := api.Hosts[origName]; has {
		delete(api.Hosts, origName)
	}
}

func (api *apiStruct) hostCheckDelete(hostNames []string) {
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

func (api *apiStruct) getMasterHostName(cpf *CallerPrefConf) string {
	api.rw.RLock()
	defer api.rw.RUnlock()

	var names []string
	for name, host := range api.Hosts {
		if host.Enable {
			names = append(names, name)
		}
	}
	return api.Caller.getPrefHostName(names, cpf)
}

func (api *apiStruct) cookieName() string {
	return apiCookieName(api.Name)
}

func loadAPIByConf(apiServer *APIServer, apiName string) (*apiStruct, error) {
	api := newAPI(apiServer, apiName)
	relName, _ := filepath.Rel(filepath.Dir(apiServer.getConfDir()), api.ConfPath)
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
	api.Hosts.init()
	log.Println(logMsg, "success")
	if api.Path == "" {
		api.Path = fmt.Sprintf("/%s/", apiName)
	}
	if !api.isValidPath(api.Path) {
		return api, fmt.Errorf("path wrong:%s", api.Path)
	}

	err = api.init()
	api.Exists = true
	return api, err
}

func (api *apiStruct) pvInc() uint64 {
	return api.apiServer.GetCounter().pvInc(api.Name)
}

func (api *apiStruct) GetPv() uint64 {
	return api.apiServer.GetCounter().GetPv(api.Name)
}

func (api *apiStruct) roomName() string {
	return fmt.Sprintf("_room_%s", api.Name)
}

func apiCookieName(apiName string) string {
	return fmt.Sprintf("%s_%s", apiPrefParamName, apiName)
}

/**
* get sorted hosts,master is at first
 */
func (api *apiStruct) getAPIHostsByReq(req *http.Request) (hs []*Host, master string, cpf *CallerPrefConf) {
	cpf = newCallerPrefConfByHTTPRequest(req, api)
	caller := api.Caller.getCallerItemByIP(cpf.GetIP())
	masterHost := api.getMasterHostName(cpf)

	hs = make([]*Host, 0)
	var hsTmp []*Host
	for _, apiHost := range api.Hosts {
		if !apiHost.Enable || caller.isHostIgnore(apiHost.Name, cpf) {
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
