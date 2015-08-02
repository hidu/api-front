package proxy

import (
	"encoding/json"
	"net/http"
	//	"fmt"
)

type BroadCastData struct {
	Id   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

func NewReqBroadCastData(req *http.Request) *BroadCastData {
	data := &BroadCastData{
		Data: make(map[string]interface{}),
	}
	data.Data["path"] = req.URL.RequestURI()
	data.Data["remote"] = req.RemoteAddr
	data.Data["method"] = req.Method
	return data
}

func (d *BroadCastData) SetData(key string, val interface{}) {
	d.Data[key] = val
}

func (d *BroadCastData) SetError(err string) {
	d.Data["err"] = err
}

func (d *BroadCastData) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}
