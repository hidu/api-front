package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
)

type portServer struct {
	Port        int
	APIServiers map[string]*APIServer
	Manager     *APIServerManager
}

// ServeHTTP serve all http request
// mutil ports
func (ps *portServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	hostInfo := strings.Split(req.Host, ":")
	host := hostInfo[0]
	apiServer := ps.getAPIServer(host)
	logMsg := fmt.Sprintf("remote=%s uri=%s refer=%s host=%s port=%d", req.RemoteAddr, req.RequestURI, req.Referer(), host, ps.Port)
	if apiServer == nil {
		dump, _ := httputil.DumpRequest(req, false)
		log.Println("[warning]", logMsg, "app not found,404,req.Host="+req.Host+","+req.URL.String(), req.Header, ",req_dump:", string(dump))
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("app not found\n----------------\npowered by api-front"))
		return
	}
	log.Println("[info]", logMsg, "server is:", apiServer.GetServerID())
	apiServer.ServeHTTP(rw, req)
}

func (ps *portServer) getAPIServer(hostName string) *APIServer {
	// 只有一个域名的情况
	if len(ps.APIServiers) == 1 {
		for _, s := range ps.APIServiers {
			return s
		}
	}
	// 域名匹配
	for _, item := range ps.APIServiers {
		if item.ServerVhostConf.hasDomain(hostName) || item.ServerVhostConf.hasDomain("default") {
			return item
		}
	}
	return nil
}

type portServerManager struct {
	PortServerMap map[int]*portServer
	manager       *APIServerManager
}

func newPortServerManager(manager *APIServerManager) *portServerManager {
	psm := &portServerManager{
		PortServerMap: make(map[int]*portServer),
		manager:       manager,
	}
	for _, signConf := range manager.mainConf.VhostConfs {
		if !signConf.Enable {
			log.Println("[warning]server ", signConf.Name, signConf.Port, " is not enable,skip")
			continue
		}
		psm.addServer(signConf)
	}
	return psm
}

// AddServer add new server
func (psm *portServerManager) addServer(itemConf *serverVhost) bool {
	if _, has := psm.PortServerMap[itemConf.Port]; !has {
		psm.PortServerMap[itemConf.Port] = &portServer{
			Port:        itemConf.Port,
			APIServiers: make(map[string]*APIServer),
			Manager:     psm.manager,
		}
	}
	ps := psm.PortServerMap[itemConf.Port]
	apiServer := newAPIServer(itemConf, psm.manager)
	log.Println("[info]add server", apiServer.serverNames())
	ps.APIServiers[apiServer.GetServerID()] = apiServer
	return true
}

func (psm *portServerManager) start() {
	var wg sync.WaitGroup
	log.Println("[info]ports total:", len(psm.PortServerMap))
	for port, ps := range psm.PortServerMap {
		wg.Add(1)
		go (func(port int, ps *portServer) {
			addr := fmt.Sprintf(":%d", port)
			log.Println(addr, "start")
			err := http.ListenAndServe(addr, ps)
			log.Println("[fatal]", addr, "exit:", err)
			wg.Done()
		})(port, ps)
	}
	wg.Wait()
	log.Println("[fatal]portServer exit")
}
