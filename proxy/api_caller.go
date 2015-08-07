package proxy

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

//var API_PREF string = "api_pref"

const (
	API_PREF_PARMA_NAME  string = "api_pref"
	API_PREF_TYPE_REQ           = "req"
	API_PREF_TYPE_COOKIE        = "cookie"
	API_PREF_TYPE_HEADER        = "header"
)

var prefTypes = []string{API_PREF_TYPE_REQ, API_PREF_TYPE_COOKIE, API_PREF_TYPE_HEADER}

type Caller []*CallerItem

type CallerItem struct {
	Note   string         `json:"note"`
	Ip     string         `json:"ip"`
	IpReg  *regexp.Regexp `json:"-"`
	Enable bool           `json:"enable"`
	Pref   []string       `json:"pref"`
	Ignore []string       `json:"ignore"`
}

func NewCaller() Caller {
	return make([]*CallerItem, 0)
}

func NewCallerItem(ip string) (*CallerItem, error) {
	item := &CallerItem{
		Ip:     ip,
		Pref:   make([]string, 0),
		Ignore: make([]string, 0),
	}
	var err error
	err = item.Init()

	return item, err
}

func (citem *CallerItem) Init() (err error) {
	citem.IpReg, err = regexp.Compile(strings.Replace(strings.Replace(citem.Ip, ".", `\.`, -1), "*", `\d+`, -1))
	if err != nil {
		log.Println("ip wrong:", citem.Ip)
	}
	if citem.Ignore == nil {
		citem.Ignore = make([]string, 0)
	}
	return err
}

func (citem *CallerItem) IsHostIgnore(host_name string) bool {
	return In_StringSlice(host_name, citem.Ignore)
}

const IP_ALL string = "*.*.*.*"

func (caller *Caller) Init() (err error) {
	has_all := false
	for _, citem := range *caller {
		err := citem.Init()
		if err != nil {
			return err
		}
		if citem.Ip == IP_ALL {
			has_all = true
		}
	}
	if !has_all {
		citem, _ := NewCallerItem(IP_ALL)
		citem.Note = "default all"
		citem.Enable = true
		citem.Init()
		caller.AddNewCallerItem(citem)
	}
	return nil
}

func (caller *Caller) GetPrefHostName(allowNames []string, cpf *CallerPrefConf) string {

	if len(allowNames) == 0 || len(*caller) == 0 {
		return StrSliceRandItem(allowNames)
	}

	for _, prefType := range prefTypes {
		if len(cpf.prefHostName[prefType]) > 0 {
			pref := StrSliceIntersectGetOne(cpf.prefHostName[prefType], allowNames)
			if pref != "" {
				return pref
			}
		}
	}
	item := caller.getCallerItemByIp(cpf.ip)
	if item != nil && len(item.Pref) > 0 {
		pref := StrSliceIntersectGetOne(item.Pref, allowNames)
		if pref != "" {
			return pref
		}
	}
	return StrSliceRandItem(allowNames)
}

func (caller Caller) Sort() {
	sort.Sort(caller)
}
func (caller Caller) Len() int {
	return len(caller)
}

/**
*让 127.0.0.1 排在127.0.0.* 前面
 */
func (caller Caller) Less(i, j int) bool {
	a_pos := strings.Index(caller[i].Ip, "*")
	if a_pos == -1 {
		return true
	}
	b_pos := strings.Index(caller[j].Ip, "*")
	if b_pos == -1 {
		return false
	}

	return a_pos > b_pos
}

func (caller Caller) Swap(i, j int) {
	caller[i], caller[j] = caller[j], caller[i]
}

var Default_Caller *CallerItem = &CallerItem{Ip: IP_ALL, Enable: true, Note: "default"}

func init() {
	Default_Caller.Init()
}

func (caller Caller) getCallerItemByIp(ip string) *CallerItem {
	for _, item := range caller {
		if !item.Enable {
			continue
		}
		if item.Ip == ip || item.IpReg.MatchString(ip) {
			return item
		}
	}
	return Default_Caller
}

func (caller *Caller) AddNewCallerItem(item *CallerItem) {
	*caller = append(*caller, item)
}

type CallerPrefConf struct {
	ip           string
	prefHostName map[string][]string
}

func (cpf *CallerPrefConf) AddNewPrefHost(prefType string, hostName string) {
	hostName = strings.TrimSpace(hostName)
	if hostName == "" {
		return
	}
	if _, has := cpf.prefHostName[prefType]; !has {
		cpf.prefHostName[prefType] = make([]string, 0)
	}
	if !In_StringSlice(hostName, cpf.prefHostName[prefType]) {
		cpf.prefHostName[prefType] = append(cpf.prefHostName[prefType], hostName)
	}
}

func (cpf *CallerPrefConf) AddNewPrefHostRaw(prefType string, str string, spitStr string) {
	strSlice := strings.Split(str, spitStr)
	for _, v := range strSlice {
		cpf.AddNewPrefHost(prefType, v)
	}
}

func (cpf *CallerPrefConf) GetIp() string {
	return cpf.ip
}

var ipReg = regexp.MustCompile(`^(\d+\.){3}\d+$`)

func NewCallerPrefConfByHttpRequest(req *http.Request, api *Api) *CallerPrefConf {
	prefConf := &CallerPrefConf{}
	prefConf.prefHostName = make(map[string][]string)

	info := strings.SplitN(req.RemoteAddr, ":", 2)
	prefConf.ip = info[0]
	x_real_ip := req.Header.Get("X-Real-Ip")
	if x_real_ip != "" && ipReg.MatchString(x_real_ip) {
		prefConf.ip = x_real_ip
	}

	//get from form data
	prefConf.AddNewPrefHostRaw(API_PREF_TYPE_REQ, req.FormValue(API_PREF_PARMA_NAME), ",")

	//get from http header
	prefConf.AddNewPrefHostRaw(API_PREF_TYPE_HEADER, req.Header.Get(API_PREF_PARMA_NAME), ",")

	//get from cookie
	cookie, err := req.Cookie(api.CookieName())
	if err == nil {
		prefConf.AddNewPrefHostRaw(API_PREF_TYPE_COOKIE, cookie.Value, ",")
	}

	return prefConf
}
