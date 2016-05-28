package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type apiStruct struct {
	ID           string       `json:"-"`
	ConfPath     string       `json:"-"`
	Path         string       `json:"path"` //api 绑定地址 前缀
	Note         string       `json:"note"`
	TimeoutMs    int          `json:"timeout_ms"`
	Hosts        Hosts        `json:"hosts"`
	Enable       bool         `json:"enable"`
	Caller       Caller       `json:"caller"`
	rw           sync.RWMutex `json:"-"`
	Exists       bool         `json:"-"`
	HostAsProxy  bool         `json:"host_as_proxy"` //是否把后端当作代理
	Pv           uint64       `json:"-"`
	LastVisit    time.Time    `json:"-"`       //最后访问时间
	Version      int64        `json:"version"` //配置文件的版本号
	apiServer    *APIServer
	Users        users        `json:"users"`
	Proxy        string       `json:"proxy"`         //使用父代理
	RespModifier RespModifier `json:"resp_modifier"` //

	proxyURL *url.URL `json:"-"` //父代理的URL object

	analysisClientNum int `json:"-"` //正在进行协议分析的客户端数量
}

// init new api for server
func newAPI(apiServer *APIServer, id string) *apiStruct {
	api := &apiStruct{
		ID:        id,
		Hosts:     newHosts(),
		apiServer: apiServer,
	}
	api.ConfPath = api.getConfPath()
	return api
}

func (api *apiStruct) getConfPath() string {
	return fmt.Sprintf("%s/%s.json", api.apiServer.getConfDir(), api.ID)
}

func (api *apiStruct) init() (err error) {
	log.Println("start load api [", api.ID, "] conf")

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

	if api.Proxy != "" {
		api.proxyURL, _ = url.Parse(api.Proxy)
	}

	if api.RespModifier == nil {
		api.RespModifier = newRespModifierSlice()
	}
	if e:=api.RespModifier.Init();e!=nil{
		return e
	}

	api.Caller.Sort()
	err = api.Caller.init()

	api.Exists = true
	return err
}

var pathReg = regexp.MustCompile(`^/([\w-/]+/?)*$`)

var apiIDReg = regexp.MustCompile(`^[\w-]+$`)

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

func (api *apiStruct) changeID(id string) error {
	if api.ID == id {
		log.Println("rename skip,not change,newName:", id)
		return nil
	}
	err := api.delete()
	if err != nil {
		return err
	}
	api.ID = id
	api.ConfPath = api.getConfPath()
	return api.save()
}

func (api *apiStruct) clone() *apiStruct {
	api.rw.RLock()
	defer api.rw.RUnlock()
	data, _ := json.Marshal(api)
	var newAPI *apiStruct
	json.Unmarshal(data, &newAPI)
	newAPI.ID = api.ID
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
	return apiCookieName(api.ID)
}

func loadAPIByConf(apiServer *APIServer, apiID string) (*apiStruct, error) {
	api := newAPI(apiServer, apiID)
	relName, _ := filepath.Rel(filepath.Dir(apiServer.getConfDir()), api.ConfPath)
	logMsg := fmt.Sprint("load api [", apiID, "],[", relName, "]")

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
		api.Path = fmt.Sprintf("/%s/", apiID)
	}
	if !api.isValidPath(api.Path) {
		return api, fmt.Errorf("path wrong:%s", api.Path)
	}

	err = api.init()
	api.Exists = true
	return api, err
}

func (api *apiStruct) pvInc() uint64 {
	return api.apiServer.GetCounter().pvInc(api.ID)
}

func (api *apiStruct) GetPv() uint64 {
	return api.apiServer.GetCounter().GetPv(api.ID)
}

func (api *apiStruct) uniqID() string {
	sc := api.apiServer.ServerVhostConf
	return fmt.Sprintf("api|%s|%d|%s", sc.Id, sc.Port, api.ID)
}

func apiCookieName(apiID string) string {
	return fmt.Sprintf("%s_%s", apiPrefParamName, apiID)
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

func (api *apiStruct) userCanEditById(id string) bool {
	if api.Users != nil && api.Users.hasUser(id) {
		return true
	}

	return api.apiServer.hasUser(id)
}

func (api *apiStruct) userCanEdit(u *User) bool {
	var id string
	if u != nil {
		id = u.ID
	}
	return api.userCanEditById(id)
}

func (api *apiStruct) analysisClientNumInc(num int) int {
	api.rw.Lock()
	defer api.rw.Unlock()
	api.analysisClientNum += num
	if api.analysisClientNum < 0 {
		api.analysisClientNum = 0
	}
	return api.analysisClientNum
}
