package proxy

import (
	"time"
	"encoding/json"
	"net/http"
)

// Host one api backend host
type Host struct {
	Name      string `json:"-"`
	URLStr    string `json:"url"`
	Enable    bool   `json:"enable"`
	Note      string `json:"note"`
	SortIndex int    `json:"sort"`
	Checked   bool   `json:"-"`
	Header   map[string]string   `json:"-"`
	HeaderStr   string   `json:"header"`
}

// Hosts api hosts
type Hosts map[string]*Host

func newHosts() Hosts {
	return make(Hosts)
}

func (h *Host) copy() *Host {
	return &Host{
		Name:      h.Name,
		URLStr:    h.URLStr,
		Enable:    h.Enable,
		Note:      h.Note,
		SortIndex: h.SortIndex,
		HeaderStr: h.HeaderStr,
	}
}

func (h *Host) init() (err error) {
	h.Header = make(map[string]string)
	hs:=make(map[string]string)
	if h.HeaderStr!=""{
		err=json.Unmarshal([]byte(h.HeaderStr), &hs)
	}
	
	for k,v:=range hs{
		h.Header[http.CanonicalHeaderKey(k)] = v
	}
	
	return
}

func (h *Host)Headers()map[string]string{
	return h.Header
}



func (hs Hosts) addNewHost(host *Host) {
	hs[host.Name] = host
}

func (hs Hosts) init() {
	for name, host := range hs {
		host.Name = name
		host.init()
	}
}


func newHost(name string, url string, enable bool) *Host {
	return &Host{
		Name:   name,
		URLStr: url,
		Enable: enable,
	}
}

func (hs Hosts) getDefaultHostName() string {
	n := time.Now().UnixNano() % int64(len(hs))
	for name := range hs {
		if n == 0 {
			return name
		}
		n = n - 1
	}
	return ""
}

func (hs Hosts) activeHostsNum() int {
	num := 0
	for _, host := range hs {
		if host.Enable {
			num = num + 1
		}
	}
	return num
}

// GetHostsWithPref tpl call this
//
func (hs Hosts) GetHostsWithPref(pref []string) []*Host {

	enableNames := []string{}
	for name, host := range hs {
		if host.Enable {
			enableNames = append(enableNames, name)
		}
	}
	var arr []*Host
	for _, name := range pref {
		if InStringSlice(name, enableNames) {
			h := hs[name].copy()
			h.Checked = true
			arr = append(arr, h)
		}
	}
	for name, host := range hs {
		if host.Enable && !InStringSlice(name, pref) {
			arr = append(arr, host)
		}
	}
	return arr
}
