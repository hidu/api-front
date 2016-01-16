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

type mainConf struct {
	ServerName  string         `json:"server_name"`
	VhostConfs  []*serverVhost `json:"vhost"`
	confPath    string         `json:"-"`
	Users       users          `json:"users"`
	Oauth2Conf  *oauth2Conf    `json:"oauth2"`
	SessionName string         `json:"session_name"` //cookie name
	SessionSk   string         `json:"session_sk"`   //secret
}

/**
*登陆方式
 */
const LOGIN_TYPE_FILE string = "file"

const LOGIN_TYPE_OAUTH string = "oauth"

func loadMainConf(confPath string) *mainConf {
	data, err := ioutil.ReadFile(confPath)
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
	fmt.Println("conf", conf)
	return conf
}

func (conf *mainConf) confDir() string {
	return filepath.Dir(conf.confPath) + string(filepath.Separator)
}

func (conf *mainConf) loadVhosts() {
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
		var item *serverVhost
		err = LoadJSONFile(fileName, &item)
		if err != nil {
			log.Println("load vhost conf [", confName, "]", "failed,err:", err)
			continue
		}
		item.SubDoamin = subDomain
		item.Port = port
		conf.VhostConfs = append(conf.VhostConfs, item)
	}
	for _, item := range conf.VhostConfs {
		if item.Users == nil {
			item.Users = NewUsers()
		}
	}
}
func (conf *mainConf) ports() (ports []int) {
	for _, item := range conf.VhostConfs {
		if !InIntSlice(item.Port, ports) {
			ports = append(ports, item.Port)
		}
	}
	return
}

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
