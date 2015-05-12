package proxy

import (
	"log"
	"math/rand"
	"os"
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
	LogFile  *os.File
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

	logPath := filepath.Dir(filepath.Dir(manager.ConfPath)) + "/log/mimo.log"
	manager.setupLog(logPath)
	defer manager.LogFile.Close()

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

func (manager *MimoServerManager) setupLog(logPath string) {
	logPathDay := logPath + "." + time.Now().Format("20060102")
	DirCheck(logPathDay)
	var err error
	manager.LogFile, err = os.OpenFile(logPathDay, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln("create log file failed [", logPathDay, "]", err)
	}
	log.SetOutput(manager.LogFile)
	SetInterval(func() {
		logPathDay := logPath + "." + time.Now().Format("20060102")
		if !File_exists(logPathDay) {
			manager.LogFile.Close()
			DirCheck(logPathDay)
			manager.LogFile, err = os.OpenFile(logPathDay, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				log.Println("create log file failed [", logPathDay, "]", err)
			}
			log.SetOutput(manager.LogFile)
		}
	}, 30)
}
