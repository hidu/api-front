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
	if !InStringSlice(hostName, cpf.prefHostName[prefType]) {
		cpf.prefHostName[prefType] = append(cpf.prefHostName[prefType], hostName)
	}
}

func (cpf *CallerPrefConf) allPrefHosts() []string {
	var hs []string
	for _, phs := range cpf.prefHostName {
		if len(phs) > 0 {
			hs = append(hs, phs...)
		}
	}
	return hs
}

func (cpf *CallerPrefConf) AddNewPrefHostRaw(prefType string, str string, spitStr string) {
	strSlice := strings.Split(str, spitStr)
	for _, v := range strSlice {
		cpf.AddNewPrefHost(prefType, v)
	}
}

func (cpf *CallerPrefConf) GetIP() string {
	return cpf.ip
}

var ipReg = regexp.MustCompile(`^(\d+\.){3}\d+$`)

func NewCallerPrefConfByHTTPRequest(req *http.Request, api *Api) *CallerPrefConf {
	prefConf := &CallerPrefConf{}
	prefConf.prefHostName = make(map[string][]string)

	info := strings.SplitN(req.RemoteAddr, ":", 2)
	prefConf.ip = info[0]
	xRealIP := strings.TrimSpace(req.Header.Get("X-Real-Ip"))
	if xRealIP != "" && ipReg.MatchString(xRealIP) {
		prefConf.ip = xRealIP
	}

	//get from form data
	prefConf.AddNewPrefHostRaw(apiPrefTypeReq, req.FormValue(apiPrefParamName), ",")

	//get from http header
	prefConf.AddNewPrefHostRaw(apiPrefTypeHeader, req.Header.Get(apiPrefParamName), ",")

	//get from cookie
	cookie, err := req.Cookie(api.cookieName())
	if err == nil {
		prefConf.AddNewPrefHostRaw(apiPrefTypeCookie, cookie.Value, ",")
	}

	return prefConf
}
