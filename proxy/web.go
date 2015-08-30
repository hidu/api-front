package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/hidu/goutils"
	"html"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// APIProxyVersion current server version
var APIProxyVersion string

func init() {
	APIProxyVersion = Assest.GetContent("/res/version")
}

type webAdmin struct {
	apiServer *APIServer
	wsServer  *socketio.Server
	wsSocket  socketio.Socket
}

func newWebAdmin(server *APIServer) *webAdmin {
	ser := &webAdmin{
		apiServer: server,
	}
	ser.wsInit()

	return ser
}
func (web *webAdmin) wsInit() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatalln("init ws server failed:", err.Error())
	}
	web.wsServer = server

	server.On("connection", func(so socketio.Socket) {
		so.Emit("hello", "hello,now:"+time.Now().String())
		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
		web.wsSocket = so
		so.Join("api_pv")
		so.On("http_analysis", func(name string) {
			api := web.apiServer.getAPIByName(name)
			if api != nil {
				err := so.Join(api.roomName())
				log.Println("join_room", api.roomName(), err)
			}
		})
	})
	server.On("error", func(so socketio.Socket) {
		log.Println("ws error:", err)
	})
	web.broadAPIPvs()
}
func (web *webAdmin) broadAPIPvs() {
	pvs := make(map[string]uint64)

	utils.SetInterval(func() {
		var pv uint64
		for name, api := range web.apiServer.Apis {
			if _, has := pvs[name]; !has {
				pvs[name] = 0
			}
			pv = api.GetPv()
			if pvs[name] == pv {
				continue
			}
			pvs[name] = pv
			data := make(map[string]interface{})
			data["name"] = name
			data["pv"] = pv
			web.wsServer.BroadcastTo("api_pv", "api_pv", data)
		}
	}, 1)
}

func (web *webAdmin) broadcastAPI(api *apiStruct, broadType string, reqData *BroadCastData) {
	roomName := api.roomName()
	log.Println("broad:", roomName, broadType)
	web.wsServer.BroadcastTo(roomName, broadType, reqData)
}

func (web *webAdmin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/_res/") {
		req.URL.Path = "/res/" + req.URL.Path[5:]
		Assest.HTTPHandler("/").ServeHTTP(rw, req)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/_socket.io/") {
		web.wsServer.ServeHTTP(rw, req)
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
	web    *webAdmin
	values map[string]interface{}
}

func (wr *webReq) execute() {
	wr.values["Title"] = "Index"
	wr.values["version"] = APIProxyVersion
	wr.values["base_url"] = "http://" + wr.req.Host
	hostInfo := strings.Split(wr.req.Host, ":")
	if hostInfo[1] == "" {
		hostInfo[1] = "80"
	}
	wr.values["req_host"] = wr.req.Host
	wr.values["host_name"] = hostInfo[0]
	port, _ := strconv.ParseInt(hostInfo[1], 10, 64)
	wr.values["host_port"] = int(port)
	wr.values["conf"] = wr.web.apiServer.ServerConf

	switch wr.req.URL.Path {
	case "/_api":
		wr.values["Title"] = "Edit"
		wr.apiEdit()
		return
	case "/_apis":
		wr.values["Title"] = "List"
		wr.apiList()
		return
	case "/_pref":
		wr.apiPref()
		return
	case "/_apipv":
		wr.apiPv()
		return
	case "/_analysis":
		wr.values["Title"] = "Analysis"
		wr.apiAnalysis()
		return
	}

	userIndexHtmlPath := wr.web.apiServer.rootConfDir() + "index.html"
	wr.values["userIndex"] = loadFile(userIndexHtmlPath)
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

	cookieName := apiCookieName(apiName)

	if prefHost != "" {
		api := wr.web.apiServer.getAPIByName(apiName)
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

func (wr *webReq) apiPv() {
	apiName := strings.TrimSpace(wr.req.FormValue("name"))
	if apiName == "" {
		wr.json(400, "param empty", nil)
		return
	}
	api := wr.web.apiServer.getAPIByName(apiName)
	if api == nil {
		wr.json(400, "api not exists", nil)
		return
	}
	wr.json(0, "suc", api.GetPv())
}
func (wr *webReq) apiAnalysis() {
	apiName := strings.TrimSpace(wr.req.FormValue("name"))
	if apiName == "" {
		wr.values["error"] = "param empty"
		wr.render("error.html", true)
		return
	}
	api := wr.web.apiServer.getAPIByName(apiName)
	if api == nil {
		wr.values["error"] = "api not exists!  <a href='/_api'>add new</a>"
		wr.render("error.html", true)
		return
	}
	wr.values["api"] = api
	wr.render("analysis.html", true)
}

func (wr *webReq) alert(msg string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s")</script>`, msg)))
}
func (wr *webReq) alertAndGo(msg string, urlstr string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s");top.location.href="%s";</script>`, msg, urlstr)))
}

// JSONResult json result when ajax call
type JSONResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (wr *webReq) json(code int, msg string, data interface{}) {
	ret := &JSONResult{code, msg, data}
	bs, _ := json.Marshal(ret)
	wr.rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	wr.rw.Write(bs)
}

func (wr *webReq) render(tplName string, layout bool) {
	html := renderHTML(tplName, wr.values, true)
	wr.rw.Write([]byte(html))
}

var webTmpName = "tpl_api_proxy"

func (wr *webReq) apiEdit() {
	req := wr.req
	name := req.FormValue("name")
	if req.Method != "POST" {
		var api *apiStruct
		if name != "" {
			apiOld := wr.web.apiServer.getAPIByName(name)
			if apiOld == nil {
				wr.values["error"] = "api not exists!  <a href='/_api'>add new</a>"
				wr.render("error.html", true)
				return
			}
			api = apiOld.clone()
		} else {
			api = newAPI(wr.web.apiServer, "")
		}
		hostsTpl := newHosts()
		hostsTpl.addNewHost(newHost(webTmpName, "http://127.0.0.1/", false))

		citem, _ := newCallerItem("")
		api.Caller.addNewCallerItem(citem)

		wr.values["api"] = &api
		wr.values["HostsTpl"] = hostsTpl
		wr.values["api_url"] = "http://" + req.Host + api.Path

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
	api := wr.web.apiServer.getAPIByName(apiName)
	apiNameOrig := req.FormValue("api_name_orig")
	apiPath := URLPathClean(req.FormValue("path"))
	if apiNameOrig == "" && api != nil {
		wr.alert("失败：新创建的模块已经存在")
		return
	}

	if !apiNameReg.MatchString(apiName) {
		wr.alert(`模块名称不满足规则：^[\w-]+$`)
		return
	}

	apiByPath := wr.web.apiServer.getAPIByPath(apiPath)

	if apiByPath != nil {
		if api == nil || (api != nil && api.Name != apiByPath.Name) {
			wr.alert(fmt.Sprintf("绑定的路径(%s)和api(%s:%s)重复", apiPath, apiByPath.Name, apiByPath.Note))
			return
		}
	}
	if api == nil {
		api = wr.web.apiServer.newAPI(apiName)
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
		api = wr.web.apiServer.newAPI(apiName)
	}

	api.Note = req.FormValue("note")
	api.TimeoutMs = int(timeout)
	api.Enable = req.FormValue("enable") == "1"
	api.Path = apiPath
	api.HostAsProxy = req.FormValue("host_as_proxy") == "1"
	if apiNameOrig != apiName {
		wr.web.apiServer.deleteAPI(apiNameOrig)
	}

	err = api.save()
	if err != nil {
		wr.alert("保存失败：" + err.Error())
		return
	}
	wr.web.apiServer.loadAPI(apiName)
	wr.alertAndGo("已经更新！", "/_api?name="+apiName)
}

func (wr *webReq) apiCallerSave() {
	req := wr.req
	apiName := req.FormValue("api_name")
	api := wr.web.apiServer.getAPIByName(apiName)
	if api == nil {
		wr.alert("api模块不存在")
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
	wr.web.apiServer.loadAPI(apiName)
	wr.json(0, "已经更新！", nil)
}

func readerHTMLInclude(fileName string) string {

	html := Assest.GetContent("/res/tpl/" + fileName)
	myfn := template.FuncMap{
		"my_include": func(name string) string {
			return readerHTMLInclude(name)
		},
	}
	tpl, _ := template.New("page_include").Delims("{%", "%}").Funcs(myfn).Parse(html)
	var bf []byte
	w := bytes.NewBuffer(bf)
	tpl.Execute(w, make(map[string]string))
	body := w.String()
	return body
}

func renderHTML(fileName string, values map[string]interface{}, layout bool) string {
	htmlStr := readerHTMLInclude(fileName)
	myfn := template.FuncMap{
		"shortTime": func(tu int64) string {
			t := time.Unix(tu, 0)
			return t.Format(timeFormatStd)
		},
		"myNum": func(n int64) string {
			if n == 0 {
				return ""
			}
			return fmt.Sprintf("%d", n)
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
		return renderHTML("layout.html", values, false)
	}
	return utils.Html_reduceSpace(body)
}
