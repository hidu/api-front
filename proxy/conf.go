package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type ServerConfItem struct {
	Port         int    `json:"port"`
	Enable       bool   `json:"enable"`
	Name         string `json:"name"`
	Note         string `json:"note"`
	HiddenCookie bool   `json:"hidden_cookie"`
}

type apiServerConf struct {
	Server []*ServerConfItem `json:"server"`
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
	return conf
}
