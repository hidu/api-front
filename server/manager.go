package server

import (
	"github.com/hidu/goutils/time_util"
	"log"
	"sync"
)

type Manager struct {
	RW      sync.RWMutex
	Servers map[int64]*APIServer
}


func (m *Manager) Init() {
	m.Servers = make(map[int64]*APIServer)
	log.Println("[trace] manager.Init,total servers:", len(m.Servers))
}

func (m *Manager) AddServerByID(id int64) error {
	log.Println("[trace] Manager.AddServerByID server_id=", id)
	m.RW.Lock()
	defer m.RW.Unlock()
	if _, has := m.Servers[id]; has {
		return nil
	}
	server, err := NewAPIServer(id)
	if err != nil {
		return err
	}
	m.Servers[id] = server
	go server.Start()
	return nil
}

func (m *Manager) Start() {
	log.Println("[trace] manager.Start Begin")
	m.LoadAllServer()
	time_util.SetInterval(func() {
		log.Println("[trace] SetInterval_30_sec_LoadAllServer")
		m.LoadAllServer()
	}, 30)
	log.Println("[trace] manager.Start END")
}

func (m *Manager) LoadAllServer() {
	ids := GetAllAPIServerIDs()
	log.Println("[trace] Manager.LoadAllServer,total=", len(ids), ids)
	var idsNew []int64
	func() {
		m.RW.RLock()
		defer m.RW.RUnlock()
		for _, server_id := range ids {
			node, has := m.Servers[server_id]
			if has {
				node.Reload()
			}else{
				idsNew = append(idsNew, server_id)
			}
		}
	}()

	for _, server_id := range idsNew {
		m.AddServerByID(server_id)
	}
}

var manager = &Manager{}

func RegisterManager() error {
	log.Println("[trace] server.RegisterManager run")
	manager.Init()
	go manager.Start()
	return nil
}
