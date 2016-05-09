package proxy

import (
	"encoding/json"
	"net/http"
	"strings"
)

// BroadCastData broad to browser data
type BroadCastData struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

func newReqBroadCastData(req *http.Request) *BroadCastData {
	data := &BroadCastData{
		Data: make(map[string]interface{}),
	}
	data.Data["request_uri"] = req.URL.RequestURI()
	remote := strings.Split(req.RemoteAddr, ":")
	data.Data["remote_last"] = remote[0]
	data.Data["method"] = req.Method
	return data
}

func (d *BroadCastData) setData(key string, val interface{}) {
	d.Data[key] = val
}

func (d *BroadCastData) setError(err string) {
	d.Data["err"] = err
}

// String json encode data
func (d *BroadCastData) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}
