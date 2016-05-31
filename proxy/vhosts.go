package proxy

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"net/http/httputil"
)

type portServer struct {
	apiServers map[int]map[string]*APIServer
	manager    *APIServerManager
}

func newPortServer(manager *APIServerManager) *portServer {
	ps := &portServer{
		apiServers: make(map[int]map[string]*APIServer),
		manager:    manager,
	}
	for _, signConf := range manager.mainConf.VhostConfs {
		if !signConf.Enable {
			log.Println("[warning]server ", signConf.Name, signConf.Port, " is not enable,skip")
			continue
		}
		ps.addServer(signConf)
	}
	return ps
}

// AddServer add new server
func (ps *portServer) addServer(itemConf *serverVhost) bool {
	apiServer := newAPIServer(itemConf, ps.manager)

	log.Println("[info]add server", apiServer.serverNames())

	if _, has := ps.apiServers[itemConf.Port]; !has {
		ps.apiServers[itemConf.Port] = make(map[string]*APIServer)
	}
	ps.apiServers[itemConf.Port][apiServer.GetServerID()] = apiServer
	return true
}

// ServeHTTP serve all http request
// mutil ports
func (ps *portServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	hostInfo := strings.Split(req.Host, ":")
	host := hostInfo[0]
	port := 80
	if len(hostInfo) == 2 {
		var err error
		port, err = strconv.Atoi(hostInfo[1])
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("bad request"))
			return
		}
	}
	apiServer := ps.getAPIServer(port, host)
	logMsg :=fmt.Sprintf("remote=%s uri=%s refer=%s host=%s port=%d",req.RemoteAddr, req.RequestURI ,req.Referer(),host,port)
	if apiServer == nil {
		dump,_:=httputil.DumpRequest(req,false)
		log.Println("[warning]", logMsg, "app not found,404,req.Host="+req.Host+","+req.URL.String(),req.Header,",req_dump:",string(dump))
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("app not found\n----------------\npowered by api-front"))
		return
	}
	log.Println("[info]", logMsg, "server is:", apiServer.serverNames())
	apiServer.ServeHTTP(rw, req)
}

func (ps *portServer) getAPIServer(port int, hostName string) *APIServer {
	as, has := ps.apiServers[port]
	if !has || len(as) == 0 {
		return nil
	}
	//只有一个域名的情况
	hasOneServer := len(as) == 1
	if hasOneServer {
		for _, s := range as {
			return s
		}
	}
	//域名匹配
	for _, item := range as {
		if item.ServerVhostConf.hasDomain(hostName) || item.ServerVhostConf.hasDomain("default") {
			return item
		}
	}
	return nil
}

func (ps *portServer) start() {
	var wg sync.WaitGroup
	log.Println("[info]ports total:", len(ps.apiServers))
	for port := range ps.apiServers {
		wg.Add(1)
		go (func(port int) {
			addr := fmt.Sprintf(":%d", port)
			log.Println(addr, "start")
			err := http.ListenAndServe(addr, ps)
			log.Println("[fatal]", addr, "exit:", err)
			wg.Done()
		})(port)
	}
	wg.Wait()
	log.Println("[fatal]portServer exit")
}
