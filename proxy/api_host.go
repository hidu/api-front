package proxy

import (
	"time"
)

type Host struct {
	Name   string `json:"-"`
	Url    string `json:"url"`
	Enable bool   `json:"enable"`
	Note   string `json:"note"`
}

type Hosts map[string]*Host

func NewHosts() Hosts {
	return make(Hosts)
}

func (hs Hosts) AddNewHost(host *Host) {
	hs[host.Name] = host
}

func (hs Hosts) Init() {
	for name, host := range hs {
		host.Name = name
	}
}

func NewHost(name string, url string, enable bool) *Host {
	return &Host{
		Name:   name,
		Url:    url,
		Enable: enable,
	}
}

func (hs Hosts) GetDefaultHostName() string {
	n := time.Now().UnixNano() % int64(len(hs))
	for name := range hs {
		if n == 0 {
			return name
		}
		n = n - 1
	}
	return ""
}
