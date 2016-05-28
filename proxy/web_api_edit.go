package proxy

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (wr *webReq) apiEdit() {
	req := wr.req
	id := req.FormValue("id")
	if req.Method != "POST" {
		var api *apiStruct
		addNew := false
		if id != "" {
			apiOld := wr.web.apiServer.getAPIByID(id)
			if apiOld == nil {
				wr.values["error"] = "api not exists!  <a href='/_/api'>add new</a>"
				wr.render("error.html", true)
				return
			}
			api = apiOld.clone()
		} else {
			api = newAPI(wr.web.apiServer, "")
			api.Enable = true
			api.TimeoutMs = 5000
			addNew = true
		}
		hostsTpl := newHosts()
		hostsTpl.addNewHost(newHost(webTmpName, "http://127.0.0.1/", true))
		if addNew {
			api.Hosts["default"] = newHost("default", "", true)
		}
		citem, _ := newCallerItem("")
		api.Caller.addNewCallerItem(citem)

		wr.values["api"] = &api
		wr.values["HostsTpl"] = hostsTpl
		wr.values["respModifiler"] = &ApiRespModifier{}
		wr.values["api_url"] = "http://" + req.Host + api.Path

		wr.values["userCanEdit"] = api.userCanEdit(wr.user)

		prefCookie, err := wr.req.Cookie(api.cookieName())
		cookiePref := ""
		if err == nil {
			cookiePref = prefCookie.Value
		}
		wr.values["cookiePref"] = strings.Split(cookiePref, ",")
		wr.values["cookiePrefStr"] = cookiePref

		wr.render("api.html", true)
		return
	}

	do := req.FormValue("do")
	switch do {
	case "base":
		wr.apiBaseSave()
	case "caller":
		wr.apiCallerSave()
	case "changeid":
		wr.apiChangeID()
	case "resp_modifier":
		wr.apiRespModifier()
	default:
		wr.alert("unknow")
	}
}

func (wr *webReq) apiChangeID() {
	req := wr.req
	origID := req.FormValue("orig_id")
	newID := req.FormValue("new_id")

	if origID == newID {
		wr.json(304, "now change", nil)
		return
	}
	if !apiIDReg.MatchString(newID) {
		wr.json(400, "name wrong", nil)
		return
	}

	origApi := wr.web.apiServer.getAPIByID(origID)
	if origApi == nil {
		wr.json(404, "api not found", nil)
		return
	}
	if !origApi.userCanEdit(wr.user) {
		wr.json(403, "没有编辑权限", nil)
		return
	}

	newApi := wr.web.apiServer.getAPIByID(newID)
	if newApi != nil {
		wr.json(404, newID+" aready exists!", nil)
		return
	}

	if err := origApi.changeID(newID); err != nil {
		wr.json(500, "rename failed", nil)
		return
	}
	wr.web.apiServer.unRegisterAPI(origID)
	wr.web.apiServer.loadAPI(newID)
	wr.json(0, "success", newID)
}

func (wr *webReq) apiBaseSave() {
	req := wr.req

	mod := req.FormValue("mod")

	if mod == "new" && !wr.web.apiServer.hasUser(wr.getUserID()) {
		wr.alert("No permissions!")
		return
	}

	timeout, err := strconv.ParseInt(req.FormValue("timeout"), 10, 64)
	if err != nil {
		wr.alert("wrong Timeout value,not int")
		return
	}
	apiID := req.FormValue("api_id")

	//绑定路径
	apiPath := URLPathClean(req.FormValue("path"))

	if !apiIDReg.MatchString(apiID) {
		wr.alert(fmt.Sprintf(`api Id not allow`, apiID))
		return
	}

	api := wr.web.apiServer.getAPIByID(apiID)
	if api != nil && mod == "new" {
		wr.alert(fmt.Sprintf(`api(%s) already exist`, apiID))
		return
	}

	if api != nil && !api.userCanEdit(wr.user) {
		wr.alert("No permissions!")
		return
	}

	//按照路径查找得到的api
	apiByPath := wr.web.apiServer.getAPIByPath(apiPath)

	if apiByPath != nil {
		if api == nil || (api != nil && api.ID != apiByPath.ID) {
			wr.alert(fmt.Sprintf("same location (%s) as api(%s:%s)", apiPath, apiByPath.ID, apiByPath.Note))
			return
		}
	}
	if api == nil {
		api = wr.web.apiServer.newAPI(apiID)
	}

	hostNames := req.PostForm["host_name"]
	hostNameOrigs := req.PostForm["host_name_orig"]
	hostUrls := req.PostForm["host_url"]
	hostNotes := req.PostForm["host_note"]
	hostEnables := req.PostForm["host_enable"]

	if len(hostNames) != len(hostUrls) || len(hostNames) != len(hostNotes) || len(hostNames) != len(hostEnables) {
		wr.alert("save failed")
		return
	}

	tmp := make(map[string]string)
	for _, val := range hostNames {
		if _, has := tmp[val]; has {
			wr.alert("别名:" + val + "，重复了")
			return
		}
	}

	for i, name := range hostNames {
		if name == "" || name == webTmpName {
			continue
		}
		host := newHost(name, hostUrls[i], true)
		host.Note = hostNotes[i]
		host.Enable = hostEnables[i] == "1"

		//		wr.web.apiServer.
		api.Hosts.addNewHost(host)
		nameOrig := hostNameOrigs[i]
		api.hostRename(nameOrig, name)
	}
	api.hostCheckDelete(hostNames)

	if api == nil {
		api = wr.web.apiServer.newAPI(apiID)
	}

	api.Note = req.FormValue("note")
	api.TimeoutMs = int(timeout)
	api.Enable = req.FormValue("enable") == "1"
	api.Path = apiPath
	api.HostAsProxy = req.FormValue("host_as_proxy") == "1"
	api.Users = make(users, 0)

	proxy := strings.TrimSpace(req.FormValue("proxy"))
	api.Proxy = proxy
	if proxy != "" {
		if api.HostAsProxy {
			wr.alert("Can not use parent proxy when run as proxy model")
			return
		}
		_u, err := url.Parse(proxy)
		if err != nil || _u.Scheme != "http" {
			wr.alert("Parent HTTP proxy wrong")
			return
		}
	}

	uids := strings.Split(req.FormValue("uids"), "|")
	if wr.user != nil {
		uids = append(uids, wr.user.ID)
	} else {
		uids = append(uids, ":any")
	}

	for _, v := range uids {
		v = strings.TrimSpace(v)
		if v != "" && !InStringSlice(v, api.Users) {
			if !wr.web.apiServer.hasUser(v) {
				api.Users = append(api.Users, v)
			}
		}
	}

	err = api.save()
	if err != nil {
		wr.alert("Save failed：" + err.Error())
		return
	}
	wr.web.apiServer.loadAPI(apiID)
	wr.alertAndGo("Save Success！", "/_/api?id="+apiID)
}

func (wr *webReq) apiCallerSave() {
	req := wr.req
	apiID := req.FormValue("api_id")
	api := wr.web.apiServer.getAPIByID(apiID)
	if api == nil {
		wr.alert("api模块不存在")
		return
	}
	if !api.userCanEdit(wr.user) {
		wr.alert("没有编辑权限")
		return
	}
	datas := req.Form["datas[]"]
	callers := newCaller()
	for _, qs := range datas {
		qv, _ := url.ParseQuery(qs)
		item, _ := newCallerItem(qv.Get("ip"))
		item.Note = qv.Get("note")
		item.Enable = qv.Get("enable") == "1"
		if qv.Get("host_names") != "" {
			item.Pref = qv["host_names"]
		}
		if qv.Get("host_ignore") != "" {
			item.Ignore = qv["host_ignore"]

			for _, ignoreName := range item.Ignore {
				if InStringSlice(ignoreName, item.Pref) {
					wr.json(1, "配置冲突("+item.IP+")\n屏蔽:"+ignoreName, nil)
					return
				}
			}
		}
		callers.addNewCallerItem(item)
	}
	api.Caller = callers

	err := api.save()
	if err != nil {
		wr.json(1, "保存配置失败:"+err.Error(), nil)
		return
	}
	wr.web.apiServer.loadAPI(apiID)
	wr.json(0, "已经更新！", nil)
}

func (wr *webReq) apiRespModifier() {
	req := wr.req
	apiID := req.FormValue("api_id")
	api := wr.web.apiServer.getAPIByID(apiID)
	if api == nil {
		wr.alert("api不存在")
		return
	}
	if !api.userCanEdit(wr.user) {
		wr.alert("没有编辑权限")
		return
	}
	datas := req.Form["datas[]"]
	ms := make([]*ApiRespModifier, 0)
	for _, qs := range datas {
		qv, _ := url.ParseQuery(qs)
		item := &ApiRespModifier{
			Note:   qv.Get("note"),
			Enable: qv.Get("enable") == "1",
			Rule:   qv.Get("rule"),
		}
		err:=item.Init()
		if(err!=nil){
			wr.json(1, "初始化失败:"+err.Error(), nil)
		}
		ms = append(ms, item)
	}
	api.RespModifier = ms

	err := api.save()
	if err != nil {
		wr.json(1, "保存配置失败:"+err.Error(), nil)
		return
	}
	wr.web.apiServer.loadAPI(apiID)
	wr.json(0, "已经更新！", nil)
}
