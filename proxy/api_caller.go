package proxy

import (
	"log"
	"regexp"
	"sort"
	"strings"
)

//var API_PREF string = "api_pref"

const (
	apiPrefParamName  string = "api_pref"
	apiPrefTypeReq           = "req"
	apiPrefTypeCookie        = "cookie"
	apiPrefTypeHeader        = "header"
)

var prefTypes = []string{apiPrefTypeReq, apiPrefTypeCookie, apiPrefTypeHeader}

type Caller []*CallerItem

type CallerItem struct {
	Note   string         `json:"note"`
	IP     string         `json:"ip"`
	IPReg  *regexp.Regexp `json:"-"`
	Enable bool           `json:"enable"`
	Pref   []string       `json:"pref"`
	Ignore []string       `json:"ignore"`
}

func newCaller() Caller {
	return make([]*CallerItem, 0)
}

func newCallerItem(ip string) (*CallerItem, error) {
	item := &CallerItem{
		IP:     ip,
		Pref:   make([]string, 0),
		Ignore: make([]string, 0),
	}
	var err error
	err = item.init()

	return item, err
}

func newCallerItemMust(ip string) *CallerItem {
	item, _ := newCallerItem(ip)
	item.Enable = true
	return item
}

func (citem *CallerItem) init() (err error) {
	citem.IPReg, err = regexp.Compile(strings.Replace(strings.Replace(citem.IP, ".", `\.`, -1), "*", `\d+`, -1))
	if err != nil {
		log.Println("ip wrong:", citem.IP)
	}
	if citem.Ignore == nil {
		citem.Ignore = make([]string, 0)
	}
	return err
}

func (citem *CallerItem) isHostIgnore(hostHame string) bool {
	return InStringSlice(hostHame, citem.Ignore)
}

const ipAll string = "*.*.*.*"

func (caller *Caller) init() (err error) {
	has_all := false
	for _, citem := range *caller {
		err := citem.init()
		if err != nil {
			return err
		}
		if citem.IP == ipAll {
			has_all = true
		}
	}
	if !has_all {
		citem := newCallerItemMust(ipAll)
		citem.Note = "default all"
		citem.Enable = true
		citem.init()
		caller.addNewCallerItem(citem)
	}
	return nil
}

func (caller *Caller) getPrefHostName(allowNames []string, cpf *CallerPrefConf) string {

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
	item := caller.getCallerItemByIP(cpf.ip)
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
	aPos := strings.Index(caller[i].IP, "*")
	if aPos == -1 {
		return true
	}
	bPos := strings.Index(caller[j].IP, "*")
	if bPos == -1 {
		return false
	}

	return aPos > bPos
}

func (caller Caller) Swap(i, j int) {
	caller[i], caller[j] = caller[j], caller[i]
}

var defaultCaller = &CallerItem{IP: ipAll, Enable: true, Note: "default"}

func init() {
	defaultCaller.init()
}

func (caller Caller) getCallerItemByIP(ip string) *CallerItem {
	for _, item := range caller {
		if !item.Enable {
			continue
		}
		if item.IP == ip || item.IPReg.MatchString(ip) {
			return item
		}
	}
	return defaultCaller
}

func (caller *Caller) addNewCallerItem(item *CallerItem) {
	*caller = append(*caller, item)
	caller.Sort()
}
