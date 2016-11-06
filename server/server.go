package server

import (
	"fmt"
	"github.com/hidu/api-front/models"
	"log"
	"net/http"
	"sync"
	"time"
)

type APIServer struct {
	Rw          sync.RWMutex
	IsRunning   bool
	StartTime   time.Time
	ReloadTime  time.Time
	ReloadTimes uint64
	ServerName  string

	ModelServerHost *models.ServerHost
	APILocations    map[int64]*APILocation

	LocationRouter *LocationRouter
}

func NewAPIServer(id int64) (*APIServer, error) {
	sm := &models.ServerHost{}
	sm.ID = id
	err := sm.Read()
	if err != nil {
		return nil, err
	}
	server := &APIServer{
		ModelServerHost: sm,
		APILocations:    make(map[int64]*APILocation),
		LocationRouter:  newLocationRouter(),
	}
	server.ServerName = fmt.Sprintf("%d_%s", sm.ID, sm.Name)
	return server, nil
}

func (sv *APIServer) LoadLocations() {
	log.Println("[trace] APIServer.LoadLocations [", sv.ServerName, "]")
	ls := sv.GetLocations()
	var addItems []*APILocation
	var delIds []int64

	sv.Rw.RLock()
	for id, item := range sv.APILocations {
		if newItem, has := ls[id]; has {
			if item.Version() != newItem.Version() {
				item.Reload()
			}
		} else {
			delIds = append(delIds, id)
		}
	}

	for id, itemNew := range ls {
		if item, has := sv.APILocations[id]; has {
			if item.Version() != itemNew.Version() {
				item.Reload()
			}
		} else {
			addItems = append(addItems, itemNew)
		}
	}
	sv.Rw.RUnlock()

	log.Println("[trace] LoadLocations,delIDs=", delIds, "addItems=", len(addItems))
	for _, id := range delIds {
		sv.DelLocationByID(id)
	}

	for _, itemNew := range addItems {
		sv.AddLocationByID(itemNew.ModelLocation.ID)
	}

}

func (sv *APIServer) GetLocations() map[int64]*APILocation {
	locations := sv.ModelServerHost.GetLocations()
	result := make(map[int64]*APILocation)
	for _, item := range locations {
		result[item.ID] = &APILocation{
			ModelLocation: item,
		}
	}
	return result
}

func (sv *APIServer) AddLocationByID(id int64) error {
	location, err := NewAPILocation(id)
	if err != nil {
		return err
	}
	sv.Rw.Lock()
	defer sv.Rw.Unlock()
	if _, has := sv.APILocations[id]; has {
		return nil
	}
	sv.APILocations[id] = location

	router := newRouterItem(id, location.ModelLocation.Location)
	sv.LocationRouter.BindRouter(router)
	return nil
}

func (sv *APIServer) DelLocationByID(id int64) error {
	sv.Rw.Lock()
	defer sv.Rw.Unlock()
	location, has := sv.APILocations[id]
	if !has {
		return nil
	}
	sv.LocationRouter.DeleteRouterByPath(location.ModelLocation.Location)
	delete(sv.APILocations, id)
	return nil
}

func (sv *APIServer) Reload() error {
	log.Println("[trace] APIServer Reload,", sv.Name())
	sv.LoadLocations()

	sv.Rw.Lock()
	defer sv.Rw.Unlock()
	sv.ReloadTime = time.Now()
	sv.ReloadTimes++
	return nil
}

func (sv *APIServer) Start() error {
	sv.Reload()
	addr := fmt.Sprintf(":%d", sv.ModelServerHost.Port)
	sv.IsRunning = true
	sv.StartTime = time.Now()
	log.Println("[trace] APIServer", addr, "start")
	err := http.ListenAndServe(addr, sv)
	log.Println("[fatal] http.ListenAndServe err", addr, err.Error())
	return err
}

func (sv *APIServer) Name() string {
	return sv.ServerName
}

func (sv *APIServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	//	router := apiServer.routers.getRouterByReqPath(req.URL.Path)
	location := sv.GetAPILocationByPath(req.URL.Path)
	if location == nil {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("location not found"))
		return
	}
	location.ServeHTTP(rw, req)

}

func (sv *APIServer) GetAPILocationByPath(p string) *APILocation {
	item := sv.LocationRouter.GetRouterItemByPath(p)
	if item == nil {
		return nil
	}
	sv.Rw.RLock()
	defer sv.Rw.RUnlock()

	if location, has := sv.APILocations[item.LocationID]; has {
		return location
	}
	return nil
}

func GetAllAPIServerIDs() []int64 {
	groupList := models.ListAllServerGroup()
	var result []int64
	for _, group := range groupList {
		hosts := group.GetServerHosts()
		for _, host := range hosts {
			result = append(result, host.ID)
		}
	}
	return result
}
