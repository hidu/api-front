package proxy

import (
	"encoding/json"
	"fmt"
	"sync"
)

type serverVhost struct {
	Id           string       `json:"id"`     //配置文件名称，全局唯一
	Group        string       `json:"group"`  //服务分组，展现用
	Domains      []string     `json:"domain"` //域名，支持多个
	Port         int          `json:"port"`
	Name         string       `json:"name"` //名称，描述信息
	Enable       bool         `json:"enable"`
	Note         string       `json:"note"`          //备注
	HiddenCookie bool         `json:"hidden_cookie"` //是否在http协议分析的时候隐藏cookie的具体值
	Users        users        `json:"users"`         //具有管理权限的用户列表
	rw           sync.RWMutex `json:"-"`
}

func (sv *serverVhost) HomeUrl(serverName string) string {
	host := serverName
	for _, name := range sv.Domains {
		if name != "default" {
			host = name
			break
		}
	}
	return fmt.Sprintf("http://%s:%d", host, sv.Port)
}

func (sv *serverVhost) String() string {
	ds, _ := json.MarshalIndent(sv, "", "  ")
	return string(ds)
}

func (sv *serverVhost) hasDomain(domain string) bool {
	for _, name := range sv.Domains {
		if name == domain {
			return true
		}
	}
	return false
}

func (sv *serverVhost) HasUser(id string) bool {
	return sv.Users != nil && sv.Users.hasUser(id)
}

func (sv *serverVhost) AddDomain(domain string) {
	sv.rw.Lock()
	defer sv.rw.Unlock()

	if !InStringSlice(domain, sv.Domains) {
		sv.Domains = append(sv.Domains, domain)
	}
}
