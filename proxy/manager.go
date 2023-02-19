package proxy

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// APIServerManager server manager
type APIServerManager struct {
	ps       *portServerManager
	ConfPath string
	LogFile  *os.File
	mainConf *mainConf
}

// NewAPIServerManager init manager
func NewAPIServerManager(confPath string) *APIServerManager {
	manager := &APIServerManager{}
	manager.mainConf = loadMainConf(confPath)
	manager.ConfPath, _ = filepath.Abs(confPath)

	manager.ps = newPortServerManager(manager)
	return manager
}

// Start start run manager
func (manager *APIServerManager) Start() {
	logPath := filepath.Dir(filepath.Dir(manager.ConfPath)) + "/log/api-front.log"
	manager.setupLog(logPath)
	defer manager.LogFile.Close()
	manager.ps.start()
	log.Println("all server shutdown")
}

func (manager *APIServerManager) rootConfDir() string {
	return filepath.Dir(manager.ConfPath) + string(filepath.Separator)
}

func (manager *APIServerManager) setupLog(logPath string) {
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
