package proxy

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type RouterItem struct {
	ApiName  string
	BindPath string
	Hander   http.HandlerFunc
}

func newRouterItem(apiName string, bindPath string, hander http.HandlerFunc) *RouterItem {
	return &RouterItem{
		ApiName:  apiName,
		BindPath: bindPath,
		Hander:   hander,
	}
}

type Routers struct {
	BindMap   map[string]*RouterItem
	BindPaths BindPathsStruct
	rw        sync.RWMutex
}

func newRouters() *Routers {
	return &Routers{
		BindMap:   make(map[string]*RouterItem),
		BindPaths: make(BindPathsStruct, 0),
	}
}

type BindPathsStruct []string

func (bs BindPathsStruct) Len() int {
	return len(bs)
}

func (bs BindPathsStruct) Less(i, j int) bool {
	return len(bs[i]) > len(bs[j])
}

func (bs BindPathsStruct) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

func (rs *Routers) String() string {
	return strings.Join(rs.BindPaths, ",")
}

func (rs *Routers) getRouterByReqPath(urlPath string) *RouterItem {
	rs.rw.RLock()
	defer rs.rw.RUnlock()

	for _, bindPath := range rs.BindPaths {
		if bindPath != "" && strings.HasPrefix(urlPath, bindPath) {
			return rs.BindMap[bindPath]
		}
	}
	return nil
}

func (rs *Routers) Sort() {
	rs.rw.Lock()
	defer rs.rw.Unlock()

	bindPaths := make(BindPathsStruct, 0, len(rs.BindMap))
	for bindPath := range rs.BindMap {
		bindPaths = append(bindPaths, bindPath)
	}
	sort.Sort(bindPaths)
	rs.BindPaths = bindPaths
	log.Println("routers_bind_path:", rs.String())
}

func (rs *Routers) deleteRouterByPath(bindPath string) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		if router, has := rs.BindMap[bindPath]; has {
			log.Println("unbind router,apiName=", router.ApiName, "bindPath=", bindPath)
			delete(rs.BindMap, bindPath)
		}
	}()
	rs.Sort()
}

func (rs *Routers) bindRouter(bindPath string, router *RouterItem) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		rs.BindMap[bindPath] = router
		log.Println("bind router,apiName=", router.ApiName, "bindPath=", bindPath)
	}()
	rs.Sort()
}
