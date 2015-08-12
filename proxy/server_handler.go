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
						rw.Write([]byte("error:" + err.Error() + "\nraw_url:" + rawUrl))
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
