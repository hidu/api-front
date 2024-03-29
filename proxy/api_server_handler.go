package proxy

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

func (apiServer *APIServer) newHandler(api *apiStruct) func(http.ResponseWriter, *http.Request) {
	bindPath := api.Path
	log.Println(apiServer.ServerVhostConf.Port, api.ID, "bind path [", bindPath, "]")
	return func(rw http.ResponseWriter, req *http.Request) {
		id := api.pvInc()
		uniqID := apiServer.uniqReqID(id)
		var broadData *BroadCastData
		needBroad := apiServer.needBroadcast(api)

		start := time.Now()

		if needBroad {
			broadData = apiServer.initBroadCastData(req)
			broadData.ID = uniqID
			broadData.setData("api_id", api.ID)
			defer func() {
				used := float64(time.Since(start).Nanoseconds()) / 1e6
				broadData.setData("used", used)
				go apiServer.broadcastAPIReq(api, broadData)
			}()
		}

		rw.Header().Set("Api-Front-Version", APIFrontVersion)
		log.Println("[access]", req.URL.String())

		relPath := req.URL.Path[len(bindPath):]
		req.Header.Set("Connection", "close")
		// add this flag,so the real backend can catch it
		req.Header.Add("Via", fmt.Sprintf("api-front/%s", APIFrontVersion))

		logData := make(map[string]any)
		var logRw sync.RWMutex

		body, err := io.ReadAll(req.Body)

		logData["body_len"] = len(body)

		if err != nil {
			rw.WriteHeader(http.StatusBadGateway)
			rw.Write([]byte("read body failed"))

			if needBroad {
				broadData.setError(err.Error())
			}
			return
		}
		// get body must by before  parse callerPref

		hosts, masterHost, cpf := api.getAPIHostsByReq(req)

		if needBroad {
			broadData.setData("master", masterHost)
			broadData.setData("remote", cpf.GetIP())
			broadData.setData("resp_status", 502) // default
		}

		_uri := req.URL.Path
		if req.URL.RawQuery != "" {
			_uri += "?" + req.URL.RawQuery
		}
		mainLogStr := fmt.Sprintf("uniqid=%s port=%d remote=%s method=%s uri=%s master=%s hostsTotal=%d refer=%s", uniqID, apiServer.ServerVhostConf.Port, req.RemoteAddr, req.Method, _uri, masterHost, len(hosts), req.Referer())

		var printLog = func(logIndex int) {
			logRw.RLock()
			defer logRw.RUnlock()
			totalUsed := fmt.Sprintf("%.3fms", float64(time.Since(start).Nanoseconds())/1e6)
			log.Println(fmt.Sprintf("[access]logindex=%d/%d", logIndex, len(hosts)), mainLogStr, fmt.Sprintf("totalUsed=%s", totalUsed), logData)
		}
		defer (func() {
			printLog(1)
		})()

		rw.Header().Set("Api-Front-Master", masterHost)

		addrInfo := strings.Split(req.RemoteAddr, ":")

		if len(hosts) == 0 {
			logData["hostTotal"] = 0
			rw.WriteHeader(http.StatusBadGateway)
			rw.Write([]byte("no backend hosts"))
			if needBroad {
				broadData.setError("no backend hosts")
			}
			return
		}

		bodyLen := int64(len(body))
		var reqs []*apiHostRequest

		// build request
		for _, apiHost := range hosts {
			isMaster := apiHost.Name == masterHost
			urlNew := ""

			serverURL := apiHost.URLStr
			if api.HostAsProxy {
				serverURL = "http://" + req.Host + api.Path
			}
			if strings.HasSuffix(urlNew, "/") {
				urlNew += strings.TrimLeft(relPath, "/")
			} else {
				urlNew += relPath
			}
			if req.URL.RawQuery != "" {
				urlNew += "?" + req.URL.RawQuery
			}

			rawURL := apiHost.URLStr + urlNew

			if isMaster {
				rw.Header().Set("Api-Front-Raw-Url", rawURL)
			}

			urlNew = serverURL + urlNew
			if needBroad {
				broadData.setData("raw_url", rawURL)
			}

			reqNew, err := http.NewRequest(req.Method, urlNew, io.NopCloser(bytes.NewReader(body)))
			if err != nil {
				log.Println("[error]build req failed:", err)
				if isMaster {
					rw.WriteHeader(http.StatusBadGateway)
					rw.Write([]byte("error:" + err.Error() + "\nraw_url:" + rawURL))
				}
				if needBroad {
					broadData.setError(err.Error())
				}
				return
			}
			copyHeaders(reqNew.Header, req.Header)

			setHeader := apiHost.Headers()
			for _k, _v := range setHeader {
				if !strings.HasPrefix(_k, "_") {
					reqNew.Header.Set(_k, _v)
				}
			}

			if _hostName, _has := setHeader[http.CanonicalHeaderKey("_host")]; _has {
				reqNew.Host = _hostName
			}

			if _cookieAppend, _has := setHeader[http.CanonicalHeaderKey("_cookie_append")]; _has {
				reqNew.Header.Set("Cookie", reqNew.Header.Get("Cookie")+"; "+_cookieAppend)
			}

			// only accept gzip encode
			acceptEncoding := reqNew.Header.Get("Accept-Encoding")
			if acceptEncoding != "" && (InStringSlice("gzip", reqNew.Header["Accept-Encoding"]) || strings.Contains(acceptEncoding, "gzip")) {
				reqNew.Header.Set("Accept-Encoding", "gzip")
			}

			if bodyLen > 0 {
				reqNew.ContentLength = bodyLen
				reqNew.Header.Set("Content-Length", fmt.Sprintf("%d", bodyLen))
			}

			if req.Header.Get("HTTP_X_FORWARDED_FOR") != "" {
				reqNew.Header.Set("HTTP_X_FORWARDED_FOR", addrInfo[0])
			}

			timeout := time.Duration(api.TimeoutMs) * time.Millisecond

			transport := &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: 0,
				}).Dial,
				TLSHandshakeTimeout: timeout,
				DisableKeepAlives:   true,
			}
			if api.HostAsProxy {
				transport.Proxy = (func(u string) func(*http.Request) (*url.URL, error) {
					return func(req *http.Request) (*url.URL, error) {
						return url.Parse(u)
					}
				})(apiHost.URLStr)
			}

			if api.proxyURL != nil {
				transport.Proxy = func(req *http.Request) (*url.URL, error) {
					return api.proxyURL, nil
				}
			}

			apiReq := &apiHostRequest{
				req:       reqNew,
				reqRaw:    req,
				transport: transport,
				apiHost:   apiHost,
				isMaster:  isMaster,
				urlNew:    urlNew,
				urlRaw:    rawURL,
				Timeout:   timeout,
			}
			reqs = append(reqs, apiReq)
		}

		// call master at first sync
		for index, apiReq := range reqs {
			if !apiReq.isMaster {
				continue
			}
			backLog := make(map[string]any)

			defer (func() {
				logRw.Lock()
				logData[fmt.Sprintf("host_%s_%d", apiReq.apiHost.Name, index)] = backLog
				logRw.Unlock()
			})()
			// 			http.DefaultClient.Do();
			hostStart := time.Now()
			backLog["isMaster"] = apiReq.isMaster
			backLog["start"] = fmt.Sprintf("%.4f", float64(hostStart.UnixNano())/1e9)
			backLog["status"] = 502 // as default

			cc := rw.(http.CloseNotifier).CloseNotify()

			go (func() {
				for range cc {
					if !apiReq.isDone {
						apiReq.transport.CancelRequest(apiReq.req)
						backLog["status"] = 499
						if needBroad {
							broadData.setData("resp_status", 499)
						}
					}
				}
			})()
			resp, err := apiReq.RoundTrip()

			if err != nil {
				log.Println("[error]call_master_sync "+apiReq.urlNew, err)
				rw.WriteHeader(http.StatusBadGateway)
				rw.Write([]byte("fetch_error:" + err.Error() + "\nraw_url:" + apiReq.urlRaw + "\nnew_url:" + apiReq.urlNew))
				if needBroad {
					broadData.setError(err.Error())
				}
				return
			}
			defer resp.Body.Close()

			// --------------------------------------------------------------
			// 修改response 数据
			_mod, _mod_err := api.RespModifier.ModifierResp(apiReq.reqRaw, resp)
			backLog["resp_mod"] = _mod
			backLog["resp_mod_err"] = _mod_err
			if _mod_err != nil {
				log.Println("[error]call_resp_mod "+apiReq.urlNew, _mod_err)
				rw.WriteHeader(http.StatusBadGateway)
				rw.Write([]byte("response modify error:" + _mod_err.Error()))

				if needBroad {
					broadData.setError(_mod_err.Error())
				}
				return
			}
			// --------------------------------------------------------------

			if needBroad {
				apiServer.addBroadCastDataResponse(broadData, resp)
			}

			for k, vs := range resp.Header {
				for _, v := range vs {
					rw.Header().Add(k, v)
				}
			}
			rw.Header().Set("Connection", "close")
			rw.WriteHeader(resp.StatusCode)
			backLog["status"] = resp.StatusCode
			n, err := io.Copy(rw, resp.Body)
			if err != nil {
				log.Println(apiReq.urlNew, "io.copy:", n, err)
			}
			hostEnd := time.Now()
			used := hostEnd.Sub(hostStart)
			backLog["end"] = fmt.Sprintf("%.4f", float64(hostEnd.UnixNano())/1e9)
			backLog["used"] = fmt.Sprintf("%.3fms", float64(used.Nanoseconds())/1e6)
		}

		if len(reqs) > 1 {
			// call other hosts async
			go (func(reqs []*apiHostRequest) {
				defer (func() {
					printLog(len(reqs))
				})()
				var wgOther sync.WaitGroup
				for index, apiReq := range reqs {
					if apiReq.isMaster {
						continue
					}
					wgOther.Add(1)
					go (func(index int, apiReq *apiHostRequest) {
						backLog := make(map[string]any)
						defer (func() {
							logRw.Lock()
							logData[fmt.Sprintf("host_%s_%d", apiReq.apiHost.Name, index)] = backLog
							logRw.Unlock()
							wgOther.Done()
						})()

						hostStart := time.Now()
						backLog["isMaster"] = apiReq.isMaster
						backLog["start"] = fmt.Sprintf("%.4f", float64(hostStart.UnixNano())/1e9)
						backLog["isMaster"] = apiReq.isMaster
						resp, err := apiReq.RoundTrip()
						if err != nil {
							log.Println("[error]call_other_async,fetch "+apiReq.urlNew, err)
							return
						}
						backLog["status"] = resp.StatusCode
						defer resp.Body.Close()

						hostEnd := time.Now()
						used := hostEnd.Sub(hostStart)
						backLog["end"] = fmt.Sprintf("%.4f", float64(hostEnd.UnixNano())/1e9)
						backLog["used"] = fmt.Sprintf("%.3fms", float64(used.Nanoseconds())/1e6)
					})(index, apiReq)
				}
				wgOther.Wait()
			})(reqs)
		}
	}
}

type apiHostRequest struct {
	req       *http.Request // 修改后的请求
	reqRaw    *http.Request // 原始的请求
	urlRaw    string
	urlNew    string
	transport *http.Transport
	apiHost   *Host
	isMaster  bool
	Timeout   time.Duration
	isDone    bool
}

func (ar *apiHostRequest) RoundTrip() (resp *http.Response, err error) {
	isTimeout := false
	time.AfterFunc(ar.Timeout, func() {
		if ar.isDone {
			return
		}
		ar.transport.CancelRequest(ar.req)
		isTimeout = true
	})
	resp, err = ar.transport.RoundTrip(ar.req)
	ar.isDone = true
	if isTimeout {
		err = fmt.Errorf("reuest timeout after:%s ", ar.Timeout)
	}
	return
}

var reqCookieDumpLine = regexp.MustCompile(`Cookie: .+\r\n`)

func (apiServer *APIServer) initBroadCastData(req *http.Request) *BroadCastData {
	data := newReqBroadCastData(req)
	dumpBody := IsRequestDumpBody(req)

	dump, _ := httputil.DumpRequest(req, dumpBody)
	reqDetail := string(dump)
	if apiServer.ServerVhostConf.HiddenCookie {
		reqDetail = reqCookieDumpLine.ReplaceAllStringFunc(reqDetail, ReqCookieHidden)
	}

	data.setData("req_detail", base64.StdEncoding.EncodeToString([]byte(reqDetail)))
	return data
}

var resCookieDumpLine = regexp.MustCompile(`Set-Cookie: .+\r\n`)

func (apiServer *APIServer) addBroadCastDataResponse(broadData *BroadCastData, resp *http.Response) {
	dumpBody := true
	ct := resp.Header.Get("Content-Type")
	broadData.setData("content-type", ct)
	if resp.StatusCode == http.StatusOK {
		dumpBody = IsContentTypeText(ct) || strings.HasPrefix(ct, "image/")
		// 内容太长的也不广播
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

	broadData.setData("resp_status", resp.StatusCode)
	dump, _ := httputil.DumpResponse(resp, false)

	resDetail := string(dump)
	if apiServer.ServerVhostConf.HiddenCookie {
		resDetail = resCookieDumpLine.ReplaceAllStringFunc(resDetail, ResCookieSetHidden)
	}
	if !dumpBody {
		resDetail += "---body skipped---"
	} else {
		bd := forgetRead(&resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			resDetail += gzipDocode(bd)
		} else {
			resDetail += bd.String()
		}
	}
	// 	fmt.Println(res_detail)
	broadData.setData("res_detail", base64.StdEncoding.EncodeToString([]byte(resDetail)))
}

func (apiServer *APIServer) broadcastAPIReq(api *apiStruct, data *BroadCastData) {
	apiServer.web.broadcastAPI(api, "req", data)

	if apiServer.needStore() {
		go (func() {
			client := &http.Client{}
			client.Timeout = 300 * time.Millisecond
			postData := make(url.Values)
			postData.Add("host_id", apiServer.ServerVhostConf.Id)
			postData.Add("data", data.String())
			_, err := client.PostForm(apiServer.manager.mainConf.StoreApiUrl, postData)
			log.Println("call_store_api,err:", err)
		})()
	}
}

// 判断是否需要将数据广播出去：有用户打开了页面在进行查看才广播
func (apiServer *APIServer) needBroadcast(api *apiStruct) bool {
	if apiServer.needStore() {
		return true
	}
	if apiServer.web.wsServer.Count() < 1 {
		return false
	}
	return api.analysisClientNum > 0
}

func (apiServer *APIServer) needStore() bool {
	if !apiServer.ServerVhostConf.StoreAble {
		return false
	}
	if apiServer.manager.mainConf.StoreApiUrl == "" {
		return false
	}
	return true
}
