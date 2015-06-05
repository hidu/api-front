package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type ServerConfItem struct {
	Port   int    `json:"port"`
	Enable bool   `json:"enable"`
	Name   string `json:"name"`
	Note   string `json:"note"`
}

type apiServerConf struct {
	Server []*ServerConfItem `json:"server"`
}

func loadServerConf(conf_path string) *apiServerConf {
	data, err := ioutil.ReadFile(conf_path)
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
