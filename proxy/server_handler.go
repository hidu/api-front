package proxy

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
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

		hosts, masterHost, cpf := api.getApiHostsByReq(req)
		broadData.SetData("master", masterHost)
		broadData.SetData("remote", cpf.GetIp())

		_uri := req.URL.Path
		if req.URL.RawQuery != "" {
			_uri += "?" + req.URL.RawQuery
		}
		mainLogStr := fmt.Sprintf("uniqid=%s port=%d remote=%s method=%s uri=%s master=%s hostsTotal=%d refer=%s", uniqId, apiServer.ServerConf.Port, req.RemoteAddr, req.Method, _uri, masterHost, len(hosts), req.Referer())

		var printLog = func(logIndex int) {
			totalUsed := fmt.Sprintf("%.3fms", float64(time.Now().Sub(start).Nanoseconds())/1e6)
			log.Println(fmt.Sprintf("logindex=%d", logIndex), mainLogStr, fmt.Sprintf("totalUsed=%s", totalUsed), logData)
		}
		defer (func() {
			printLog(0)
		})()

		rw.Header().Set("Api-Proxy-Master", masterHost)

		addrInfo := strings.Split(req.RemoteAddr, ":")

		if len(hosts) == 0 {
			logData["hostTotal"] = 0
			rw.WriteHeader(http.StatusBadGateway)
			rw.Write([]byte("no backend hosts"))
			broadData.SetError("no backend hosts")
			return
		}

		bodyLen := int64(len(body))
		reqs := make([]*apiHostRequest, 0)

		//build request
		for _, api_host := range hosts {
			isMaster := api_host.Name == masterHost
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

			if isMaster {
				rw.Header().Set("Api-Proxy-Raw-Url", rawUrl)
			}

			urlNew = serverUrl + urlNew

			broadData.SetData("raw_url", rawUrl)

			reqNew, err := http.NewRequest(req.Method, urlNew, ioutil.NopCloser(bytes.NewReader(body)))
			if err != nil {
				log.Println("build req failed:", err)
				if isMaster {
					rw.WriteHeader(http.StatusBadGateway)
					rw.Write([]byte("error:" + err.Error() + "\nraw_url:" + rawUrl))
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
				transport.Proxy = (func(u string) func(*http.Request) (*url.URL, error) {
					return func(req *http.Request) (*url.URL, error) {
						return url.Parse(u)
					}
				})(api_host.Url)
			}
			apiReq := &apiHostRequest{
				req:       reqNew,
				transport: transport,
				apiHost:   api_host,
				isMaster:  isMaster,
				urlNew:    urlNew,
				urlRaw:    rawUrl,
			}
			reqs = append(reqs, apiReq)
		}

		//call master at first sync
		for index, apiReq := range reqs {
			if !apiReq.isMaster {
				continue
			}
			backLog := make(map[string]interface{})
			logData[fmt.Sprintf("host_%s_%d", apiReq.apiHost.Name, index)] = backLog
			hostStart := time.Now()
			backLog["isMaster"] = apiReq.isMaster
			backLog["start"] = fmt.Sprintf("%.4f", float64(hostStart.UnixNano())/1e9)
			resp, err := apiReq.transport.RoundTrip(apiReq.req)
			if err != nil {
				log.Println("fetch "+apiReq.urlNew, err)
				rw.WriteHeader(http.StatusBadGateway)
				rw.Write([]byte("fetch_error:" + err.Error() + "\nraw_url:" + apiReq.urlRaw + "\nnew_url:" + apiReq.urlNew))
				return
			}
			defer resp.Body.Close()

			apiServer.addBroadCastDataResponse(broadData, resp)

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

		//call other hosts async
		go (func(reqs []*apiHostRequest) {
			defer (func() {
				printLog(1)
			})()
			var wgOther sync.WaitGroup
			for index, apiReq := range reqs {
				if apiReq.isMaster {
					continue
				}
				wgOther.Add(1)
				go (func(index int, apiReq *apiHostRequest) {
					defer wgOther.Done()
					backLog := make(map[string]interface{})
					logData[fmt.Sprintf("host_%s_%d", apiReq.apiHost.Name, index)] = backLog
					hostStart := time.Now()
					backLog["isMaster"] = apiReq.isMaster
					backLog["start"] = fmt.Sprintf("%.4f", float64(hostStart.UnixNano())/1e9)
					backLog["isMaster"] = apiReq.isMaster
					resp, err := apiReq.transport.RoundTrip(apiReq.req)
					if err != nil {
						log.Println("fetch "+apiReq.urlNew, err)
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

type apiHostRequest struct {
	req       *http.Request
	urlRaw    string
	urlNew    string
	transport *http.Transport
	apiHost   *Host
	isMaster  bool
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

	data.SetData("req_detail", base64.StdEncoding.EncodeToString([]byte(req_detail)))
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
	broadData.SetData("res_detail", base64.StdEncoding.EncodeToString([]byte(res_detail)))
}

func (apiServer *ApiServer) BroadcastApiReq(api *Api, data *BroadCastData) {
	apiServer.web.BroadcastApi(api, "req", data)
}
