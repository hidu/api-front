package proxy

import (
	//	"fmt"
	//	"log"
	//	"net/url"
	//	"strings"
	"time"
)

type ApiPath struct {
	Path      string   `json:"-"`
	Note      string   `json:"note"`
	HostNames []string `json:"hosts"`
	Enable    bool     `json:"enable"`
}

func (api_path *ApiPath) init() error {
	return nil
}

func NewApiPath(pathStr string, note string) *ApiPath {
	back := &ApiPath{
		Path:   pathStr,
		Note:   note,
		Enable: true,
	}
	return back
}

func (api_path *ApiPath) GetMasterHostName() string {
	hlen := len(api_path.HostNames)
	if hlen < 1 {
		return ""
	}
	if hlen == 1 {
		return api_path.HostNames[0]
	}
	n := time.Now().UnixNano() % int64(hlen)
	return api_path.HostNames[n]
}

type ApiPathMap map[string]*ApiPath

func NewApiPathMap() ApiPathMap {
	return make(ApiPathMap)
}

func (path_map *ApiPathMap) init() error {
	return nil
}

func (path_map ApiPathMap) RegirestNewPath(api_path *ApiPath) {
	path_map[api_path.Path] = api_path
}

func (path_map ApiPathMap) HostRename(orig_name, new_name string) {
	for _, api_path := range path_map {
		for i, host_name_str := range api_path.HostNames {
			if host_name_str == orig_name {
				api_path.HostNames[i] = new_name
			}
		}
	}
}

func (path_map ApiPathMap) HostCheckDelete(host_name_map map[string]int) {
	for _, api_path := range path_map {
		host_name_slice := make([]string, 0)
		has_change := false
		for _, host_name_str := range api_path.HostNames {
			if _, has := host_name_map[host_name_str]; has {
				host_name_slice = append(host_name_slice, host_name_str)
			} else {
				has_change = true
			}
		}
		if has_change {
			api_path.HostNames = host_name_slice
		}
	}
}

func (path_map ApiPathMap) PathRename(path_orig, path_new string) {
	if path_orig == "" || path_orig == path_new {
		return
	}
	if _, has := path_map[path_orig]; has {
		delete(path_map, path_orig)
	}
}
