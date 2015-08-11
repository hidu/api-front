package proxy

import (
	"time"
)

type Host struct {
	Name      string `json:"-"`
	Url       string `json:"url"`
	Enable    bool   `json:"enable"`
	Note      string `json:"note"`
	SortIndex int    `json:"sort"`
	Checked   bool   `json:"-"`
}

type Hosts map[string]*Host

func NewHosts() Hosts {
	return make(Hosts)
}

func (h *Host) Copy() *Host {
	return &Host{
		Name:      h.Name,
		Url:       h.Url,
		Enable:    h.Enable,
		Note:      h.Note,
		SortIndex: h.SortIndex,
	}
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

func (hs Hosts) ActiveHostsNum() int {
	num := 0
	for _, host := range hs {
		if host.Enable {
			num = num + 1
		}
	}
	return num
}

func (hs Hosts) GetHostsWithPref(pref []string) []*Host {

	enableNames := []string{}
	for name, host := range hs {
		if host.Enable {
			enableNames = append(enableNames, name)
		}
	}
	arr := make([]*Host, 0)
	for _, name := range pref {
		if In_StringSlice(name, enableNames) {
			h := hs[name].Copy()
			h.Checked = true
			arr = append(arr, h)
		}
	}
	for name, host := range hs {
		if host.Enable && !In_StringSlice(name, pref) {
			arr = append(arr, host)
		}
	}
	return arr
}
