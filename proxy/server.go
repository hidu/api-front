package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ApiServer struct {
	Enable     bool
	Apis       map[string]*Api
	manager    *ApiServerManager
	ConfDir    string
	Rw         sync.RWMutex
	routers    *Routers
	web        *WebAdmin
	ServerConf *ServerConfItem
	counter    *Counter
}

func NewApiServer(conf *ServerConfItem, manager *ApiServerManager) *ApiServer {
	apiServer := &ApiServer{ServerConf: conf, manager: manager}
	apiServer.ConfDir = fmt.Sprintf("%s/api_%d", filepath.Dir(manager.ConfPath), conf.Port)
	apiServer.Apis = make(map[string]*Api)
	apiServer.routers = NewRouters()
	apiServer.web = NewWebAdmin(apiServer)
	apiServer.counter = NewCounter(apiServer)
	return apiServer
}

func (apiServer *ApiServer) Start() error {
	addr := fmt.Sprintf(":%d", apiServer.ServerConf.Port)

	apiServer.loadAllApis()
	log.Println("start server:", addr)
	err := http.ListenAndServe(addr, apiServer)
	return err
}

func (apiServer *ApiServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	router := apiServer.routers.GetRouterByReqPath(req.URL.Path)
	if router != nil {
		router.Hander.ServeHTTP(rw, req)
		return
	}

	if strings.HasPrefix(req.URL.Path, "/_") || req.URL.Path == "/" {
		apiServer.web.ServeHTTP(rw, req)
	} else {
		http.Error(rw, "Api Not Found (api-proxy)", http.StatusNotFound)
	}
}

func (apiServer *ApiServer) loadAllApis() {
	fileNames, _ := filepath.Glob(apiServer.ConfDir + "/*.json")
	for _, fileName := range fileNames {
		log.Println("start load conf file:", fileName)

		baseName := filepath.Base(fileName)

		apiName := baseName[:len(baseName)-5]

		if strings.HasPrefix(apiName, "_") {
			continue
		}

		apiServer.loadApi(apiName)
	}
}

func (apiServer *ApiServer) deleteApi(apiName string) {
	apiServer.Rw.Lock()
	defer apiServer.Rw.Unlock()
	api, has := apiServer.Apis[apiName]
	if !has {
		return
	}
	api.Delete()
	delete(apiServer.Apis, apiName)
}

func (apiServer *ApiServer) newApi(apiName string) *Api {
	return NewApi(apiServer, apiName)
}

func (apiServer *ApiServer) GetConfDir() string {
	return apiServer.ConfDir
}

func (apiServer *ApiServer) loadApi(apiName string) error {
	apiServer.Rw.Lock()
	defer apiServer.Rw.Unlock()

	api, err := LoadApiByConf(apiServer, apiName)
	if err != nil {
		log.Println("load api failed,", apiName, err)
		return err
	}

	log.Printf("load api [%s] success", apiName)

	apiServer.Apis[apiName] = api
	if api.Enable {
		router := NewRouterItem(apiName, api.Path, apiServer.newHandler(api))
		apiServer.routers.BindRouter(api.Path, router)
	} else {
		apiServer.routers.DeleteRouterByPath(api.Path)
		log.Printf("api [%s] is not enable,skip", apiName)
	}

	return nil
}

func (apiServer *ApiServer) GetUniqReqId(id uint64) string {
	return fmt.Sprintf("%s_%d", time.Now().Format("20060102_150405"), id)
}

func (apiServer *ApiServer) newHandler(api *Api) func(http.ResponseWriter, *http.Request) {
	bindPath := api.Path
	log.Println(apiServer.ServerConf.Port, api.Name, "bind path [", bindPath, "]")
	return func(rw http.ResponseWriter, req *http.Request) {
		id := api.PvInc()
		uniqId := apiServer.GetUniqReqId(id)

		broadData := apiServer.initBroadCastData(req)
		broadData.Id = uniqId
		start := time.Now()
		defer func() {
			used := float64(time.Now().Sub(start).Nanoseconds()) / 1e6
			broadData.SetData("used", used)
			go apiServer.BroadcastApiReq(api, broadData)
		}()

		rw.Header().Set("Api-Proxy-Version", API_PROXY_VERSION)
		log.Println(req.URL.String())

		relPath := req.URL.Path[len(bindPath):]
		req.Header.Set("Connection", "close")

		logData := make(map[string]interface{})

		body, err := ioutil.ReadAll(req.Body)

		logData["body_len"] = len(body)

		if err != nil {
			rw.WriteHeader(http.StatusBadGateway)
			rw.Write([]byte("read body failed"))
			broadData.SetError(err.Error())
			return
		}
		//get body must by before  parse callerPref

		hosts, masterHost := api.getApiHostsByReq(req)
		broadData.SetData("master", masterHost)

		defer (func() {
			uri := req.URL.Path
			if req.URL.RawQuery != "" {
				uri += "?" + req.URL.RawQuery
			}
			log.Println(apiServer.ServerConf.Port, req.RemoteAddr, req.Method, uri, "master:", masterHost, logData)
		})()

		rw.Header().Set("Api-Proxy-Master", masterHost)

		addrInfo := strings.Split(req.RemoteAddr, ":")

		if len(hosts) == 0 {
			logData["hostTotal"] = 0
			rw.WriteHeader(http.StatusBadGateway)
			rw.Write([]byte("Api has No Backend Hosts"))
			broadData.SetError("no backend hosts")
			return
		}

		bodyLen := int64(len(body))
		var wg sync.WaitGroup

		for index, api_host := range hosts {
			wg.Add(1)
			go (func(api_host *Host, rw http.ResponseWriter, req *http.Request, index int) {
				defer wg.Done()

				start := time.Now()
				isMaster := api_host.Name == masterHost
				backLog := make(map[string]interface{})
				logData[fmt.Sprintf("host_%s_%d", api_host.Name, index)] = backLog

				backLog["isMaster"] = isMaster

				urlNew := ""

				serverUrl := api_host.Url
				if api.HostAsProxy {
					serverUrl = "http://" + req.Host + api.Path
				}
				if strings.HasSuffix(urlNew, "/") {
					urlNew += strings.TrimLeft(relPath, "/")
				} else {
					urlNew += relPath
				}
				if req.URL.RawQuery != "" {
					urlNew += "?" + req.URL.RawQuery
				}

				rawUrl := api_host.Url + urlNew

				urlNew = serverUrl + urlNew

				backLog["raw_url"] = rawUrl

				broadData.SetData("raw_url", rawUrl)

				reqNew, err := http.NewRequest(req.Method, urlNew, ioutil.NopCloser(bytes.NewReader(body)))
				if err != nil {
					log.Println("build req failed:", err)
					if isMaster {
						rw.WriteHeader(http.StatusBadGateway)
						rw.Write([]byte("apiServer error:" + err.Error() + "\nraw_url:" + rawUrl))
					}
					broadData.SetError(err.Error())
					return

				}
				copyHeaders(reqNew.Header, req.Header)

				//only accept gzip encode
				accept_encoding := reqNew.Header.Get("Accept-Encoding")
				if accept_encoding != "" && (In_StringSlice("gzip", reqNew.Header["Accept-Encoding"]) || strings.Contains(accept_encoding, "gzip")) {
					reqNew.Header.Set("Accept-Encoding", "gzip")
				}

				if bodyLen > 0 {
					reqNew.ContentLength = bodyLen
					reqNew.Header.Set("Content-Length", fmt.Sprintf("%d", bodyLen))
				}

				reqNew.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])

				transport := &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					Dial: (&net.Dialer{
						Timeout:   time.Duration(api.TimeoutMs) * time.Millisecond,
						KeepAlive: 30 * time.Second,
					}).Dial,
					TLSHandshakeTimeout: 10 * time.Second,
				}
				if api.HostAsProxy {
					transport.Proxy = func(req *http.Request) (*url.URL, error) {
						return url.Parse(api_host.Url)
					}
				}

				resp, err := transport.RoundTrip(reqNew)

				if isMaster {
					rw.Header().Set("Api-Proxy-Raw-Url", rawUrl)
				}

				if err != nil {
					log.Println("fetch "+urlNew, err)
					if isMaster {
						rw.WriteHeader(http.StatusBadGateway)
						rw.Write([]byte("apiServer error:" + err.Error() + "\nraw_url:" + rawUrl))
					}
					return
				}
				defer resp.Body.Close()
				if isMaster {

					apiServer.addBroadCastDataResponse(broadData, resp)

					for k, vs := range resp.Header {
						for _, v := range vs {
							rw.Header().Add(k, v)
						}
					}
					rw.Header().Set("Connection", "close")
					rw.WriteHeader(resp.StatusCode)
					n, err := io.Copy(rw, resp.Body)
					if err != nil {
						log.Println(urlNew, "io.copy:", n, err)
					}

				}

				used := time.Now().Sub(start)
				backLog["used"] = fmt.Sprintf("%.3f ms", float64(used.Nanoseconds())/1e6)
			})(api_host, rw, req, index)
		}
		wg.Wait()

	}
}

var reqCookieDumpLine = regexp.MustCompile(`Cookie: .+\r\n`)

func (apiServer *ApiServer) initBroadCastData(req *http.Request) *BroadCastData {
	data := NewReqBroadCastData(req)
	dumpBody := IsRequestDumpBody(req)

	dump, _ := httputil.DumpRequest(req, dumpBody)
	req_detail := string(dump)
	if apiServer.ServerConf.HiddenCookie {
		req_detail = reqCookieDumpLine.ReplaceAllStringFunc(req_detail, ReqCookieHidden)
	}

	data.SetData("req_detail", req_detail)
	return data
}

var resCookieDumpLine = regexp.MustCompile(`Set-Cookie: .+\r\n`)

func (apiServer *ApiServer) addBroadCastDataResponse(broadData *BroadCastData, resp *http.Response) {
	dumpBody := true
	if resp.StatusCode == http.StatusOK {
		dumpBody = IsContentTypeText(resp.Header.Get("Content-Type"))
		if dumpBody && resp.ContentLength > 0 && resp.ContentLength > 1e6 {
			dumpBody = false
		}
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
	case http.StatusNotFound:
	case http.StatusSeeOther:
	case http.StatusForbidden:
	case http.StatusGone:
	case http.StatusBadGateway:
	case http.StatusInternalServerError:
	case http.StatusServiceUnavailable:
	case http.StatusGatewayTimeout:
		dumpBody = true
		break
	default:
		break
	}

	broadData.SetData("resp_status", resp.StatusCode)
	dump, _ := httputil.DumpResponse(resp, false)

	res_detail := string(dump)
	if apiServer.ServerConf.HiddenCookie {
		res_detail = resCookieDumpLine.ReplaceAllStringFunc(res_detail, ResCookieSetHidden)
		//		res_detail = resCookieDumpLine.ReplaceAllString(res_detail, "Set-Cookie: ------hidden------\r\n")
	}
	if !dumpBody {
		res_detail += "---body skipped---"
	} else {
		bd := forgetRead(&resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			res_detail += gzipDocode(bd)
		} else {
			res_detail += bd.String()
		}
	}
	//	fmt.Println(res_detail)
	broadData.SetData("res_detail", res_detail)
}

func (apiServer *ApiServer) BroadcastApiReq(api *Api, data *BroadCastData) {
	apiServer.web.BroadcastApi(api, "req", data)
}

func (apiServer *ApiServer) getApiByName(name string) *Api {
	if api, has := apiServer.Apis[name]; has {
		return api
	}
	return nil
}

func (apiServer *ApiServer) getApiByPath(bindPath string) *Api {
	bindPath = UrlPathClean(bindPath)
	for _, api := range apiServer.Apis {
		if api.Path == bindPath {
			return api
		}
	}
	return nil
}

func (apiServer *ApiServer) GetCounter() *Counter {
	return apiServer.counter
}
