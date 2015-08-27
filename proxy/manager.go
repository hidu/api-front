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

type ApiServerManager struct {
	Servers    map[int]*ApiServer
	ConfPath   string
	LogFile    *os.File
	ServerConf *apiServerConf
}

func NewApiServerManager(confPath string) *ApiServerManager {
	manager := &ApiServerManager{}
	manager.Servers = make(map[int]*ApiServer)
	serConf := loadServerConf(confPath)

	manager.ServerConf = serConf
	manager.ConfPath, _ = filepath.Abs(confPath)

	for _, signConf := range serConf.Server {
		if !signConf.Enable {
			log.Println("server ", signConf.Name, signConf.Port, " is not enable,skip")
			continue
		}
		manager.AddServer(signConf)
	}
	return manager
}

func (manager *ApiServerManager) AddServer(conf *ServerConfItem) bool {
	mimo := NewApiServer(conf, manager)
	if _, has := manager.Servers[conf.Port]; has {
		log.Println("ignore add server port:", conf.Port)
		return false
	}
	log.Println("add server port:", conf.Port)
	manager.Servers[conf.Port] = mimo
	return true
}

func (manager *ApiServerManager) Start() {
	logPath := filepath.Dir(filepath.Dir(manager.ConfPath)) + "/log/api-man.log"
	manager.setupLog(logPath)
	defer manager.LogFile.Close()

	var wg sync.WaitGroup
	for _, mimo := range manager.Servers {
		wg.Add(1)
		go (func(mimo *ApiServer) {
			mimo.Start()
			wg.Done()
		})(mimo)
	}
	wg.Wait()
	log.Println("all server shutdown")
}

func (manager *ApiServerManager) setupLog(logPath string) {
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
		if !FileExists(logPathDay) {
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
