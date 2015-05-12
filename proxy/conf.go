package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type mimoServerConf struct {
	Server []struct {
		Port   int  `json:"port"`
		Enable bool `json:"enable"`
	} `json:"server"`
}

func loadServerConf(conf_path string) *mimoServerConf {
	data, err := ioutil.ReadFile(conf_path)
	if err != nil {
		log.Fatalln(err)
	}
	var conf *mimoServerConf
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalln(err)
	}
	return conf
}
