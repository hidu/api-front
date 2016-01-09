package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
)

type apiServerConf struct {
	ServerName string            `json:"server_name"`
	Server     []*serverConfItem `json:"server"`
	confPath   string            `json:"-"`
	Users      users             `json:"users"`
	Oauth2Conf *oauth2Conf       `json:"oauth2"`
}

type serverConfItem struct {
	Port         int    `json:"port"`
	Enable       bool   `json:"enable"`
	Name         string `json:"name"`
	Note         string `json:"note"`
	HiddenCookie bool   `json:"hidden_cookie"`
	SubDoamin    string `json:"sub_domain"`
	Users        users  `json:"users"`
}

func loadServerConf(confPath string) *apiServerConf {
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalln(err)
	}
	var conf *apiServerConf
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
	fmt.Println("conf", conf)
	return conf
}

func (conf *apiServerConf) confDir() string {
	return filepath.Dir(conf.confPath) + string(filepath.Separator)
}

func (conf *apiServerConf) loadVhosts() {
	vhostConfDir := conf.confDir() + "vhost" + string(filepath.Separator)
	fileNames, err := filepath.Glob(vhostConfDir + "*.json")
	if err != nil {
		log.Println("load vhost conf failed:", err)
		return
	}
	log.Println("vhost files total:", len(fileNames))

	vhostNameReg := regexp.MustCompile(`(([a-z][a-z0-9]*)_)?(\d+)\.json`)
	for _, fileName := range fileNames {
		_, confName := filepath.Split(fileName)
		if confName == "" {
			continue
		}
		matchs := vhostNameReg.FindStringSubmatch(confName)
		if len(matchs) < 1 {
			log.Println("skip vhost conf:", confName)
			continue
		}
		subDomain := matchs[2]
		port, err := strconv.Atoi(matchs[3])
		if err != nil || port > 65535 {
			log.Println("skip vhost conf:", confName, ",port wrong")
			continue
		}
		var item *serverConfItem
		err = LoadJSONFile(fileName, &item)
		if err != nil {
			log.Println("load vhost conf [", confName, "]", "failed,err:", err)
			continue
		}
		item.SubDoamin = subDomain
		item.Port = port
		conf.Server = append(conf.Server, item)
	}
	for _, item := range conf.Server {
		if item.Users == nil {
			item.Users = NewUsers()
		}
	}
}
func (conf *apiServerConf) ports() (ports []int) {
	for _, item := range conf.Server {
		if !InIntSlice(item.Port, ports) {
			ports = append(ports, item.Port)
		}
	}
	return
}

func (conf *apiServerConf) parseOauthConf() {
	if conf.Oauth2Conf == nil {
		return
	}
	conf.Oauth2Conf.checkConf()
}

func (conf *apiServerConf) String() string {
	ds, _ := json.MarshalIndent(conf, "", "  ")
	return string(ds)
}
