package proxy

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

	log.Println("[info]add server", apiServer.serverName())

	if _, has := ps.apiServers[itemConf.Port]; !has {
		ps.apiServers[itemConf.Port] = make(map[string]*APIServer)
	}
	ps.apiServers[itemConf.Port][apiServer.subDomain()] = apiServer
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
	logMsg := req.RemoteAddr + " " + req.RequestURI + " " + req.Referer()
	if apiServer == nil {
		log.Println("[warning]", logMsg, "app not found,404")
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("app not found\n----------------\npowered by api-front"))
		return
	}
	log.Println("[info]", logMsg, "server is:", apiServer.serverName())
	apiServer.ServeHTTP(rw, req)
}

func (ps *portServer) getAPIServer(port int, hostName string) *APIServer {
	as, has := ps.apiServers[port]
	if !has || len(as) == 0 {
		return nil
	}
	hasOneServer := len(as) == 1
	//no server name or has just one server
	//return the default server
	if ps.manager.mainConf.ServerName == "" || hasOneServer {
		for _, item := range as {
			domain := item.subDomain()
			if domain == "" {
				return item
			}
		}
		if !hasOneServer {
			return nil
		}
	}
	if !strings.HasSuffix(hostName, "."+ps.manager.mainConf.ServerName) {
		return nil
	}
	i := len(hostName) - len(ps.manager.mainConf.ServerName) - 1
	subDoamin := hostName[:i]
	server, has := as[subDoamin]
	if !has {
		return nil
	}
	return server
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
