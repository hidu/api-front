package server

import (
	"github.com/hidu/api-front/models"
	"log"
	"net/http"
)

type APILocation struct {
	ModelLocation *models.Location
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
	}, nil
}

func (l *APILocation) PathMatch(pathStr string) {

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
	return err
}

func (l *APILocation) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte(l.ModelLocation.Location))
}
