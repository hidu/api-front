package server

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
)

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

type LocationRouterItem struct {
	LocationID int64
	BindPath   string
}

func (lr *LocationRouterItem) String() string {
	return fmt.Sprintf("location_id=%d,location=%s", lr.LocationID, lr.BindPath)
}

func newRouterItem(locationID int64, bindPath string) *LocationRouterItem {
	return &LocationRouterItem{
		LocationID: locationID,
		BindPath:   bindPath,
	}
}

type LocationRouter struct {
	BindMap   map[string]*LocationRouterItem
	BindPaths bindPathsStruct
	rw        sync.RWMutex
}

func newLocationRouter() *LocationRouter {
	return &LocationRouter{
		BindMap:   make(map[string]*LocationRouterItem),
		BindPaths: make(bindPathsStruct, 0),
	}
}

func (rs *LocationRouter) String() string {
	return strings.Join(rs.BindPaths, ",")
}

func (rs *LocationRouter) GetRouterItemByPath(urlPath string) *LocationRouterItem {
	if strings.HasPrefix(urlPath, "/_") {
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
func (rs *LocationRouter) Sort() {
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

func (rs *LocationRouter) DeleteRouterByPath(bindPath string) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		if router, has := rs.BindMap[bindPath]; has {
			log.Println("unbind router ", router)
			delete(rs.BindMap, bindPath)
		}
	}()
	rs.Sort()
}

func (rs *LocationRouter) BindRouter(router *LocationRouterItem) {
	func() {
		rs.rw.Lock()
		defer rs.rw.Unlock()
		rs.BindMap[router.BindPath] = router
		log.Println("bind router", router)
	}()
	rs.Sort()
}
