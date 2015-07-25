package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hidu/goutils"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var API_PROXY_VERSION string

func init() {
	API_PROXY_VERSION = Assest.GetContent("/res/version")
}

type WebAdmin struct {
	apiServer *ApiServer
}

func NewWebAdmin(mimo *ApiServer) *WebAdmin {
	ser := &WebAdmin{
		apiServer: mimo,
	}
	return ser
}

func (web *WebAdmin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/_res/") {
		req.URL.Path = "/res/" + req.URL.Path[5:]
		Assest.HttpHandler("/").ServeHTTP(rw, req)
		return
	}

	wr := &webReq{
		rw:     rw,
		req:    req,
		web:    web,
		values: make(map[string]interface{}),
	}
	wr.execute()
}

type webReq struct {
	rw     http.ResponseWriter
	req    *http.Request
	web    *WebAdmin
	values map[string]interface{}
}

func (wr *webReq) execute() {
	wr.values["version"] = API_PROXY_VERSION
	wr.values["base_url"] = "http://" + wr.req.Host
	wr.values["server_list"] = wr.web.apiServer.manager.ServerConf.Server
	host_info := strings.Split(wr.req.Host, ":")
	if host_info[1] == "" {
		host_info[1] = "80"
	}
	wr.values["req_host"]=wr.req.Host
	wr.values["host_name"] = host_info[0]
	port, _ := strconv.ParseInt(host_info[1], 10, 64)
	wr.values["host_port"] = int(port)
	wr.values["conf"] = wr.web.apiServer.ServerConf

	if wr.req.URL.Path == "/_api" {
		wr.apiEdit()
		return
	}
	if wr.req.URL.Path == "/_apis" {
		wr.apiList()
		return
	}
	if wr.req.URL.Path == "/_pref" {
		wr.apiPref()
		return
	}
	wr.render("index.html", true)
}

func (wr *webReq) apiList() {
	wr.values["apis"] = wr.web.apiServer.Apis
	wr.render("list.html", true)
}
func (wr *webReq) apiPref() {
	apiName := strings.TrimSpace(wr.req.FormValue("name"))
	prefHost := strings.TrimSpace(wr.req.FormValue("host"))

	if apiName == "" {
		wr.json(400, "param empty", nil)
		return
	}

	cookieName := ApiCookieName(apiName)

	if prefHost != "" {
		api := wr.web.apiServer.getApiByName(apiName)
		if api == nil {
			wr.json(400, "api not exists", nil)
			return
		}
	}

	cookie := &http.Cookie{Name: cookieName, Value: prefHost, Path: "/"}
	if prefHost != "" {
		cookie.Expires = time.Now().AddDate(1, 0, 0)
		cookie.MaxAge = 86400 * 365
	} else {
		cookie.MaxAge = -1
	}

	http.SetCookie(wr.rw, cookie)

	wr.json(0, "success", prefHost)
}

func (wr *webReq) alert(msg string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s")</script>`, msg)))
}
func (wr *webReq) alertAndGo(msg string, urlstr string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s");top.location.href="%s";</script>`, msg, urlstr)))
}

type JsonResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (wr *webReq) json(code int, msg string, data interface{}) {
	ret := &JsonResult{code, msg, data}
	bs, _ := json.Marshal(ret)
	wr.rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	wr.rw.Write(bs)
}

func (wr *webReq) render(tplName string, layout bool) {
	html := render_html(tplName, wr.values, true)
	wr.rw.Write([]byte(html))
}

var web_tmp_name = "tpl_api_proxy"

func (wr *webReq) apiEdit() {
	req := wr.req
	name := req.FormValue("name")
	if req.Method != "POST" {
		var api *Api
		if name != "" {
			apiOld := wr.web.apiServer.getApiByName(name)
			if apiOld == nil {
				wr.values["error"] = "api not exists!  <a href='/_api'>add new</a>"
				wr.render("error.html", true)
				return
			}
			api = apiOld.Clone()
		} else {
			api = NewApi(wr.web.apiServer.ConfDir, "")
		}
		hostsTpl := NewHosts()
		hostsTpl.AddNewHost(NewHost(web_tmp_name, "http://127.0.0.1/", false))

		citem, _ := NewCallerItem("")
		api.Caller.AddNewCallerItem(citem)

		wr.values["api"] = api
		wr.values["HostsTpl"] = hostsTpl
		wr.values["api_url"] = "http://" + req.Host + api.Path

		prefCookie, err := wr.req.Cookie(api.CookieName())
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

	}
}
func (wr *webReq) apiBaseSave() {
	req := wr.req
	timeout, err := strconv.ParseInt(req.FormValue("timeout"), 10, 64)
	if err != nil {
		wr.alert("超时时间错误,不是int")
		return
	}

	apiName := req.FormValue("api_name")
	api := wr.web.apiServer.getApiByName(apiName)
	apiNameOrig := req.FormValue("api_name_orig")
	apiPath := UrlPathClean(req.FormValue("path"))
	if apiNameOrig == "" && api != nil {
		wr.alert("失败：新创建的模块已经存在")
		return
	}

	if !ApiNameReg.MatchString(apiName) {
		wr.alert(`模块名称不满足规则：^[\w-]+$`)
		return
	}

	apiByPath := wr.web.apiServer.getApiByPath(apiPath)

	if apiByPath != nil {
		if api == nil || (api != nil && api.Name != apiByPath.Name) {
			wr.alert(fmt.Sprintf("绑定的路径(%s)和api(%s:%s)重复", apiPath, apiByPath.Name, apiByPath.Note))
			return
		}
	}
	if api == nil {
		api = wr.web.apiServer.newApi(apiName)
	}

	hostNames := req.PostForm["host_name"]
	hostNameOrigs := req.PostForm["host_name_orig"]
	hostUrls := req.PostForm["host_url"]
	hostNotes := req.PostForm["host_note"]
	hostEnables := req.PostForm["host_enable"]

	if len(hostNames) != len(hostUrls) || len(hostNames) != len(hostNotes) || len(hostNames) != len(hostEnables) {
		wr.alert("保存失败：数据格式错误")
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
		if name == "" || name == web_tmp_name {
			continue
		}
		host := NewHost(name, hostUrls[i], true)
		host.Note = hostNotes[i]
		host.Enable = hostEnables[i] == "1"

		//		wr.web.apiServer.
		api.Hosts.AddNewHost(host)
		nameOrig := hostNameOrigs[i]
		api.HostRename(nameOrig, name)
	}
	api.HostCheckDelete(hostNames)

	if api == nil {
		api = wr.web.apiServer.newApi(apiName)
	}

	api.Note = req.FormValue("note")
	api.TimeoutMs = int(timeout)
	api.Enable = req.FormValue("enable") == "1"
	api.Path = apiPath
	api.HostAsProxy = req.FormValue("host_as_proxy") == "1"

	if apiNameOrig != apiName {
		wr.web.apiServer.deleteApi(apiNameOrig)
	}

	err = api.Save()
	if err != nil {
		wr.alert("保存失败：" + err.Error())
		return
	}
	wr.web.apiServer.loadApi(apiName)
	wr.alertAndGo("已经更新！", "/_api?name="+apiName)
}

func (wr *webReq) apiCallerSave() {
	req := wr.req
	apiName := req.FormValue("api_name")
	api := wr.web.apiServer.getApiByName(apiName)
	if api == nil {
		wr.alert("api模块不存在")
		return
	}
	datas := req.Form["datas[]"]
	callers := NewCaller()
	for _, qs := range datas {
		qv, _ := url.ParseQuery(qs)
		item, _ := NewCallerItem(qv.Get("ip"))
		item.Note = qv.Get("note")
		item.Enable = qv.Get("enable") == "1"
		if qv.Get("host_names") != "" {
			item.Pref = qv["host_names"]
		}
		if qv.Get("host_ignore") != "" {
			item.Ignore = qv["host_ignore"]

			for _, ignoreName := range item.Ignore {
				if In_StringSlice(ignoreName, item.Pref) {
					wr.json(1, "配置冲突("+item.Ip+")\n屏蔽:"+ignoreName, nil)
					return
				}
			}
		}
		callers.AddNewCallerItem(item)
	}
	api.Caller = callers

	err := api.Save()
	if err != nil {
		wr.json(1, "保存配置失败:"+err.Error(), nil)
		return
	}
	wr.web.apiServer.loadApi(apiName)
	wr.json(0, "已经更新！", nil)
}

func reader_html_include(fileName string) string {

	html := Assest.GetContent("/res/tpl/" + fileName)
	myfn := template.FuncMap{
		"my_include": func(name string) string {
			return reader_html_include(name)
		},
	}
	tpl, _ := template.New("page_include").Delims("{%", "%}").Funcs(myfn).Parse(html)
	var bf []byte
	w := bytes.NewBuffer(bf)
	tpl.Execute(w, make(map[string]string))
	body := w.String()
	return body
}

func render_html(fileName string, values map[string]interface{}, layout bool) string {
	htmlStr := reader_html_include(fileName)
	myfn := template.FuncMap{
		"shortTime": func(tu int64) string {
			t := time.Unix(tu, 0)
			return t.Format(TIME_FORMAT_STD)
		},
		"myNum": func(n int64) string {
			if n == 0 {
				return ""
			} else {
				return fmt.Sprintf("%d", n)
			}
		},
		"in_array": func(name string, names []string) bool {
			for _, v := range names {
				if v == name {
					return true
				}
			}
			return false
		},
		"str_eq": func(x, y interface{}) bool {
			ret := fmt.Sprintf("%x", x) == fmt.Sprintf("%x", y)
			return ret
		},
		"my_include": func(fileName string) string {
			return "include (" + fileName + ") with Delims {%my_include %}"
		},
		"h": func(str string) string {
			return html.EscapeString(str)
		},
	}

	tpl, _ := template.New("page").Funcs(myfn).Parse(htmlStr)

	var bf []byte
	w := bytes.NewBuffer(bf)
	tpl.Execute(w, values)
	body := w.String()
	if layout {
		values["body"] = body
		return render_html("layout.html", values, false)
	}
	return utils.Html_reduceSpace(body)
}
