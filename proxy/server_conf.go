package proxy

import (
	"encoding/json"
	// "fmt"
	"log"
	"os"
	"path/filepath"
)

type mainConf struct {
	VhostConfs   []*serverVhost `json:"-"`
	confPath     string
	Users        users       `json:"users"`
	Oauth2Conf   *oauth2Conf `json:"oauth2"`
	SessionName  string      `json:"session_name"` // cookie name
	SessionSk    string      `json:"session_sk"`   // secret
	PortRange    *PortRange  `json:"port_range"`
	StoreApiUrl  string      `json:"store_api_url"`
	StoreViewUrl string      `json:"store_view_url"`
}

type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

/**
*登陆方式
 */
const LOGIN_TYPE_FILE string = "file"

const LOGIN_TYPE_OAUTH string = "oauth"

func loadMainConf(confPath string) *mainConf {
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatalln(err)
	}
	var conf *mainConf
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalln(err)
	}
	conf.confPath, _ = filepath.Abs(confPath)
	if conf.Users == nil {
		conf.Users = NewUsers()
	}
	conf.loadVhosts()
	conf.parseOauthConf()
	log.Println(confPath, conf)
	return conf
}

func (conf *mainConf) confDir() string {
	return filepath.Dir(conf.confPath) + string(filepath.Separator)
}

func (conf *mainConf) loadVhosts() {
	vhostConfDir := conf.confDir() + "vhost" + string(filepath.Separator)
	fileNames, err := filepath.Glob(vhostConfDir + string(filepath.Separator) + "*.json")
	if err != nil {
		log.Println("load vhost conf failed:", err)
		return
	}
	log.Println("vhost files total:", len(fileNames))

	for _, fileName := range fileNames {
		_, confName := filepath.Split(fileName)
		log.Println("start load file ", fileName, "fileName=", confName)
		if confName == "" {
			continue
		}
		var item *serverVhost
		err = LoadJSONFile(fileName, &item)
		if err != nil {
			log.Println("load vhost conf [", confName, "]", "failed,err:", err)
			continue
		}

		if item.Port > 65535 || item.Port < 1 {
			log.Println("skip vhost conf:", confName, ",port wrong")
			continue
		}
		if item.Domains == nil {
			item.Domains = []string{}
		}
		if item.Users == nil {
			item.Users = NewUsers()
		}
		item.Id = confName[:len(confName)-len(".json")]
		conf.VhostConfs = append(conf.VhostConfs, item)
		log.Println("loaded vhosts file:", fileName, item)
	}
	for _, item := range conf.VhostConfs {
		if item.Users == nil {
			item.Users = NewUsers()
		}
	}
}

// func (conf *mainConf) ports() (ports []int) {
// 	for _, item := range conf.VhostConfs {
// 		if !InIntSlice(item.Port, ports) {
// 			ports = append(ports, item.Port)
// 		}
// 	}
// 	return
// }

func (conf *mainConf) parseOauthConf() {
	if conf.Oauth2Conf == nil {
		return
	}
	conf.Oauth2Conf.checkConf()
}

func (conf *mainConf) String() string {
	ds, _ := json.MarshalIndent(conf, "", "  ")
	return string(ds)
}

func (conf *mainConf) userLoginType() string {
	if conf.Oauth2Conf != nil && conf.Oauth2Conf.Enable {
		return LOGIN_TYPE_OAUTH
	}
	return LOGIN_TYPE_FILE
}
