package proxy

import (
	"regexp"
	"strings"
)

var domainReg = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+$`)

func (wr *webReq) vhostInfo() {
	if wr.req.Method == "GET" {
		wr.values["vhost"] = wr.web.apiServer.ServerVhostConf
		wr.values["userCanEdit"] = wr.user != nil && (wr.web.apiServer.hasUser(wr.user.ID))
		wr.render("vhost.html", true)
	} else if wr.req.Method == "POST" {
		// 		id:=wr.req.FormValue("id")
		// 		port:=wr.req.FormValue("port")
		vhost := wr.web.apiServer.ServerVhostConf
		vhost.Name = wr.req.FormValue("name")
		vhost.Note = wr.req.FormValue("note")

		domain_arr := strings.Split(wr.req.FormValue("domain"), "\n")

		var domains []string
		for _, line := range domain_arr {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !domainReg.MatchString(line) {
				wr.alert("域名错误：" + line)
				return
			}
			domains = append(domains, line)
		}
		vhost.Domains = domains
		vhost.Enable = true
	}
}
