package proxy

import (
	"net/http"
	"regexp"
	"strings"
)

type CallerPrefConf struct {
	ip           string
	prefHostName map[string][]string
}

func (cpf *CallerPrefConf) AddNewPrefHost(prefType string, hostName string) {
	hostName = strings.TrimSpace(hostName)
	if hostName == "" {
		return
	}
	if _, has := cpf.prefHostName[prefType]; !has {
		cpf.prefHostName[prefType] = make([]string, 0)
	}
	if !In_StringSlice(hostName, cpf.prefHostName[prefType]) {
		cpf.prefHostName[prefType] = append(cpf.prefHostName[prefType], hostName)
	}
}

func (cpf *CallerPrefConf) AddNewPrefHostRaw(prefType string, str string, spitStr string) {
	strSlice := strings.Split(str, spitStr)
	for _, v := range strSlice {
		cpf.AddNewPrefHost(prefType, v)
	}
}

func (cpf *CallerPrefConf) GetIp() string {
	return cpf.ip
}

var ipReg = regexp.MustCompile(`^(\d+\.){3}\d+$`)

func NewCallerPrefConfByHttpRequest(req *http.Request, api *Api) *CallerPrefConf {
	prefConf := &CallerPrefConf{}
	prefConf.prefHostName = make(map[string][]string)

	info := strings.SplitN(req.RemoteAddr, ":", 2)
	prefConf.ip = info[0]
	x_real_ip := strings.TrimSpace(req.Header.Get("X-Real-Ip"))
	if x_real_ip != "" && ipReg.MatchString(x_real_ip) {
		prefConf.ip = x_real_ip
	}

	//get from form data
	prefConf.AddNewPrefHostRaw(API_PREF_TYPE_REQ, req.FormValue(API_PREF_PARMA_NAME), ",")

	//get from http header
	prefConf.AddNewPrefHostRaw(API_PREF_TYPE_HEADER, req.Header.Get(API_PREF_PARMA_NAME), ",")

	//get from cookie
	cookie, err := req.Cookie(api.CookieName())
	if err == nil {
		prefConf.AddNewPrefHostRaw(API_PREF_TYPE_COOKIE, cookie.Value, ",")
	}

	return prefConf
}
