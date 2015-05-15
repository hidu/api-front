package proxy

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"log"
)

type RouterItem struct {
	ApiName string
	BindPath   string
	Hander     http.HandlerFunc
}

func NewRouterItem(apiName string, bindPath string, hander http.HandlerFunc) *RouterItem {
	return &RouterItem{
		ApiName: apiName,
		BindPath:   bindPath,
		Hander:     hander,
	}
}

type Routers struct {
	BindMap   map[string]*RouterItem
	BindPaths BindPathsStruct
}

func NewRouters() *Routers {
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
	return strings.Join(rs.BindPaths, "\n")
}

func (rs *Routers) GetRouterByReqPath(url_path string) *RouterItem {
	for _, bind_path := range rs.BindPaths {
		if bind_path != "" && strings.HasPrefix(url_path, bind_path) {
			return rs.BindMap[bind_path]
		}
	}
	return nil
}

func (rs *Routers) Sort() {
	bindPaths := make(BindPathsStruct, 0, len(rs.BindMap))
	for bind_path := range rs.BindMap {
		bindPaths = append(bindPaths, bind_path)
	}
	sort.Sort(bindPaths)
	rs.BindPaths = bindPaths
	fmt.Println(rs.String(), "----->\n")
}

func (rs *Routers) DeleteRouterByPath(bind_path string) {
	if router, has := rs.BindMap[bind_path]; has {
		log.Println("unbind router,apiName=",router.ApiName,"bindPath=",bind_path)
		delete(rs.BindMap, bind_path)
	}
	rs.Sort()
}

func (rs *Routers) BindRouter(bind_path string, router *RouterItem) {
	rs.BindMap[bind_path] = router
	log.Println("bind router,apiName=",router.ApiName,"bindPath=",bind_path)
	rs.Sort()
}
