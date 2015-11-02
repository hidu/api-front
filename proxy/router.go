package proxy

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type routerItem struct {
	APIName  string
	BindPath string
	Hander   http.HandlerFunc
}

func newRouterItem(apiName string, bindPath string, hander http.HandlerFunc) *routerItem {
	return &routerItem{
		APIName:  apiName,
		BindPath: bindPath,
		Hander:   hander,
	}
}

type routers struct {
	BindMap   map[string]*routerItem
	BindPaths bindPathsStruct
	rw        sync.RWMutex
}

func newRouters() *routers {
	return &routers{
		BindMap:   make(map[string]*routerItem),
		BindPaths: make(bindPathsStruct, 0),
	}
}

type bindPathsStruct []string

// Len len
func (bs bindPathsStruct) Len() int {
	return len(bs)
}

// Less sort func
func (bs bindPathsStruct) Less(i, j int) bool {
	return len(bs[i]) > len(bs[j])
}

func (bs bindPathsStruct) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

func (rs *routers) String() string {
	return strings.Join(rs.BindPaths, ",")
}

func (rs *routers) getRouterByReqPath(urlPath string) *routerItem {
	if strings.HasPrefix(urlPath,"/_"){
		return nil
	}
	rs.rw.RLock()
	defer rs.rw.RUnlock()
	for _, bindPath := range rs.BindPaths {
		if bindPath != "" && strings.HasPrefix(urlPath, bindPath) {
			return rs.BindMap[bindPath]
		}
	}
	return nil
}

// Sort by name
func (rs *routers) Sort() {
	rs.rw.Lock()
	defer rs.rw.Unlock()

	bindPaths := make(bindPathsStruct, 0, len(rs.BindMap))
	for bindPath := range rs.BindMap {
		bindPaths = append(bindPaths, bindPath)
	}
	sort.Sort(bindPaths)
	rs.BindPaths = bindPaths
	log.Println("routers_bind_path:", rs.String())
}

func (rs *routers) deleteRouterByPath(bindPath string) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		if router, has := rs.BindMap[bindPath]; has {
			log.Println("unbind router,apiName=", router.APIName, "bindPath=", bindPath)
			delete(rs.BindMap, bindPath)
		}
	}()
	rs.Sort()
}


func (rs *routers) bindRouter(bindPath string, router *routerItem) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		rs.BindMap[bindPath] = router
		log.Println("bind router,apiName=", router.APIName, "bindPath=", bindPath)
	}()
	rs.Sort()
}
