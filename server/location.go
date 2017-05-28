package server

import (
	"github.com/hidu/api-front/models"
	"log"
	"net/http"
	"sync"
)

type APILocation struct {
	ModelLocation *models.Location
	APIBackends map[int64]*APIBackend
	Rw sync.RWMutex
}

func NewAPILocation(id int64) (*APILocation, error) {
	lm := &models.Location{}
	lm.ID = id
	err := lm.Read()
	if err != nil {
		return nil, err
	}
	return &APILocation{
		ModelLocation: lm,
		APIBackends:make(map[int64]*APIBackend),
	}, nil
}


func (l *APILocation) IsEnable() bool {
	if l.ModelLocation == nil {
		return false
	}
	
	return l.ModelLocation.Status == STATUS_OK
}

func (l *APILocation) Version() int64 {
	return l.ModelLocation.UpdateTime.UnixNano()
}

func (l *APILocation) Reload() error {
	log.Println("[trace] APILocation.Reload", l.ModelLocation)
	err := l.ModelLocation.Read()
	if err != nil {
		l.ModelLocation = nil
	}
	l.LoadBackends()
	return err
}

func (l *APILocation)LoadBackends(){
	log.Println("[trace] APILocation.LoadBackends [", l.ModelLocation.ID, "]")
	backends:=l.GetBackends()
	var addItems []*APIBackend
	var delIds []int64
	
	l.Rw.RLock()
	
	for id, item := range l.APIBackends {
		if newItem, has := backends[id]; has {
			if item.Version() != newItem.Version() {
				item.Reload()
			}
		} else {
			delIds = append(delIds, id)
		}
	}

	for id, itemNew := range backends {
		if item, has := l.APIBackends[id]; has {
			if item.Version() != itemNew.Version() {
				item.Reload()
			}
		} else {
			addItems = append(addItems, itemNew)
		}
	}
	l.Rw.RUnlock()

	log.Println("[trace] LoadBackends,delIDs=", delIds, "addItems=", len(addItems))
	for _, id := range delIds {
		l.DelBackendByID(id)
	}

	for _, itemNew := range addItems {
		l.AddBackendByID(itemNew.GetBackendID())
	}
}

func (l *APILocation)DelBackendByID(id int64) (error){
	l.Rw.Lock()
	defer l.Rw.Unlock()
	bk,has:=l.APIBackends[id]
	if !has{
		return nil
	}
	bk.Stop()
	delete(l.APIBackends, id)
	return nil
}

func (l *APILocation)AddBackendByID(id int64) (error){
	bk,err:=NewApiBackend(id)
	if err!=nil{
		return err
	}
	l.Rw.Lock()
	defer l.Rw.Unlock()
	
	if _,has:=l.APIBackends[id];has{
		return nil
	}
	if err:=bk.Reload();err!=nil{
		return err
	}
	
	l.APIBackends[id] = bk
	log.Println("AddBackendByID:",id,bk.ModelBackEnd.URLStr)
	
	return nil
}


func (l *APILocation)GetBackends() (map[int64]*APIBackend){
	backends:=l.ModelLocation.GetBackends()
	result := make(map[int64]*APIBackend)
	for _, item := range backends {
		result[item.ID] = NewAPIBackendByMod(item)
	}
	return result
}


func (l *APILocation) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	bindPath:=l.ModelLocation.Location
	relPath := req.URL.Path[len(bindPath):]
	l.Rw.RLock()
	defer l.Rw.RUnlock()
	
    for _,ab:=range l.APIBackends{
		ab.ServeHTTP(rw, req,relPath)
		break
    }
    
    if len(l.APIBackends)<1{
	    rw.WriteHeader(http.StatusNotFound)
	    rw.Write([]byte("404 Not Found\n"))
	    rw.Write([]byte("no backend url"))
    }
}
