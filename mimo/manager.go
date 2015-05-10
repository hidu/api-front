package mimo

import (
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"time"
)

var randR *rand.Rand

func init() {
	randR = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type MimoServerManager struct {
	Servers  map[int]*MimoServer
	ConfPath string
}

func NewMimoServerManager(conf_path string) *MimoServerManager {
	manager := &MimoServerManager{}
	manager.Servers = make(map[int]*MimoServer)
	serConf := loadServerConf(conf_path)

	manager.ConfPath, _ = filepath.Abs(conf_path)

	for _, signConf := range serConf.Server {
		if !signConf.Enable {
			continue
		}
		manager.AddServer(signConf.Port)
	}
	return manager
}

func (manager *MimoServerManager) AddServer(port int) bool {
	mimo := NewMimoServer(port, manager)
	if _, has := manager.Servers[port]; has {
		log.Println("ignore add server port:", port)
		return false
	}
	log.Println("add server port:", port)
	manager.Servers[port] = mimo
	return true
}

func (manager *MimoServerManager) Start() {
	var wg sync.WaitGroup
	for _, mimo := range manager.Servers {
		wg.Add(1)
		go (func(mimo *MimoServer) {
			mimo.Start()
			wg.Done()
		})(mimo)
	}
	wg.Wait()
}
