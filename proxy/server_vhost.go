package proxy

import (
	"fmt"
)

type serverVhost struct {
	Group        string `json:"group"`
	SubDoamin    string `json:"sub_domain"`
	Port         int    `json:"port"`
	Name         string `json:"name"`
	Enable       bool   `json:"enable"`
	Note         string `json:"note"`
	HiddenCookie bool   `json:"hidden_cookie"`
	Users        users  `json:"users"`
}

func (sv *serverVhost) HomeUrl(serverName string) string {
	host := serverName
	if sv.SubDoamin != "" {
		host = fmt.Sprintf("%s.%s", sv.SubDoamin, serverName)
	}
	return fmt.Sprintf("http://%s:%d", host, sv.Port)
}
