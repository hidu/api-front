package proxy

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

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
		citem.Note = "all"
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
	//若请求指定了
	if len(cpf.PrefHostName) > 0 {
		pref := StrSliceIntersectGetOne(cpf.PrefHostName, allowNames)
		if pref != "" {
			return pref
		}
	}
	item := caller.getCallerItemByIp(cpf.Ip)
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
	Ip           string
	PrefHostName []string
}

func (cpf *CallerPrefConf) AddNewPrefHost(hostName string) {
	hostName = strings.TrimSpace(hostName)
	if hostName == "" {
		return
	}
	if !In_StringSlice(hostName, cpf.PrefHostName) {
		cpf.PrefHostName = append(cpf.PrefHostName, hostName)
	}
}

var API_PREF string = "api_pref"

func NewCallerPrefConfByHttpRequest(req *http.Request) *CallerPrefConf {
	perfConf := &CallerPrefConf{}
	perfConf.PrefHostName = make([]string, 0)

	info := strings.SplitN(req.RemoteAddr, ":", 2)
	perfConf.Ip = info[0]
	tmpSlice := strings.Split(req.FormValue(API_PREF), ",")
	for _, itemStr := range tmpSlice {
		perfConf.AddNewPrefHost(itemStr)
	}

	headerTmpSlice := strings.Split(req.Header.Get(API_PREF), ",")
	for _, itemStr := range headerTmpSlice {
		perfConf.AddNewPrefHost(itemStr)
	}

	return perfConf
}
