package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/gorilla/sessions"
	"github.com/hidu/goutils/time_util"
)

// APIFrontVersion current server version
var APIFrontVersion string
var sessionName string = "apifront"

func init() {
	APIFrontVersion = strings.TrimSpace(string(Asset.GetContent("/resource/version")))
}

type webAdmin struct {
	apiServer    *APIServer
	wsServer     *socketio.Server
	wsSocket     socketio.Socket
	userConf     *usersConf
	sessionStore *sessions.CookieStore
}

func newWebAdmin(server *APIServer) *webAdmin {
	ser := &webAdmin{
		apiServer: server,
	}
	ser.wsInit()
	ser.userConf = loadUsers(filepath.Join(server.rootConfDir(), "users"))

	sname := server.manager.mainConf.SessionName
	if sname != "" {
		sessionName = sname
	}

	cookie_sk := "something-very-secret/" + server.manager.mainConf.userLoginType() + server.manager.mainConf.SessionSk

	ser.sessionStore = sessions.NewCookieStore([]byte(cookie_sk))

	ser.sessionStore.Options.Path = "/_/"
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

		analysisAPIs := make(map[string]*apiStruct)

		so.On("disconnection", func() {
			log.Println("on disconnect", so.Id())
			for id, api := range analysisAPIs {
				api.analysisClientNumInc(-1)
				delete(analysisAPIs, id)
			}
		})
		web.wsSocket = so
		so.Join("api_pv")
		so.On("http_analysis", func(api_id string) {
			api := web.apiServer.getAPIByID(api_id)
			log.Println("socket.io on http_analysis", api_id)
			if api != nil {
				uniqID := api.uniqID()

				if _, has := analysisAPIs[uniqID]; !has {
					err := so.Join(uniqID)
					log.Println("join_room", uniqID, err)
					analysisAPIs[uniqID] = api
					api.analysisClientNumInc(1)
				}

				msg := make(map[string]any)
				msg["client_num"] = api.analysisClientNum
				msg["api_id"] = api.ID

				so.Emit("s_http_analysis", msg)
			}
		})
	})
	time_util.SetInterval(func() {
		web.wsServer.BroadcastTo("api_pv", "hello", "hello,now:"+time.Now().String())
	}, 30)

	server.On("error", func(so socketio.Socket) {
		log.Println("ws error:", err)
	})
	web.broadAPIPvs()
}

func (web *webAdmin) broadAPIPvs() {
	pvs := make(map[string]uint64)

	time_util.SetInterval(func() {
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
			data := make(map[string]any)
			data["name"] = name
			data["pv"] = pv
			web.wsServer.BroadcastTo("api_pv", "api_pv", data)
		}
	}, 1)
}

func (web *webAdmin) broadcastAPI(api *apiStruct, broadType string, reqData *BroadCastData) {
	roomName := api.uniqID()
	log.Println("broad:", roomName, broadType)
	web.wsServer.BroadcastTo(roomName, broadType, reqData)
}

func (web *webAdmin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	web.serveHTTP(rw, req)
	// if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") || strings.HasPrefix(req.URL.Path, "/_socket.io/") {
	// 	web.serveHTTP(rw, req)
	// 	return
	// }
	// rw.Header().Set("Content-Encoding", "gzip")
	// gz := gzip.NewWriter(rw)
	// defer gz.Close()
	// gzr := gzipResponseWriter{Writer: gz, ResponseWriter: rw}
	// web.serveHTTP(gzr, req)
}

func (web *webAdmin) serveHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/_/res/") {
		http.StripPrefix("/_/res/", Asset.HTTPHandler("/resource/")).ServeHTTP(rw, req)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/_socket.io/") {
		web.wsServer.ServeHTTP(rw, req)
		return
	}

	session, _ := web.sessionStore.Get(req, sessionName)

	wr := &webReq{
		rw:      rw,
		req:     req,
		web:     web,
		values:  make(map[string]any),
		session: session,
	}
	wr.execute()
}

type webReq struct {
	rw      http.ResponseWriter
	req     *http.Request
	web     *webAdmin
	values  map[string]any
	user    *User
	session *sessions.Session
}

func (wr *webReq) execute() {
	wr.values["Title"] = "Index"
	wr.values["version"] = APIFrontVersion
	wr.values["base_url"] = "http://" + wr.req.Host
	hostInfo := strings.Split(wr.req.Host, ":")
	if hostInfo[1] == "" {
		hostInfo[1] = "80"
	}
	wr.values["req_host"] = wr.req.Host
	wr.values["host_name"] = hostInfo[0]

	// 	wr.values["server_name"] = serverName

	port, _ := strconv.ParseInt(hostInfo[1], 10, 64)
	wr.values["host_port"] = int(port)
	wr.values["conf"] = wr.web.apiServer.ServerVhostConf
	// 	wr.session.Values["aaa"] = "aaa"
	wr.getUser()

	// 		fmt.Println("session", wr.session.Values)
	wr.values["isLogin"] = wr.user != nil
	wr.values["user"] = wr.user
	if wr.user != nil {
		wr.values["uname"] = wr.user.DisplayName()
	}

	// 	if wr.req.Method == "POST" && wr.req.URL.Path != "/_login" && wr.user == nil {
	// 		wr.alert("login required")
	// 		return
	// 	}
	userIndexHTMLPath := wr.web.apiServer.rootConfDir() + "index.html"
	wr.values["userIndex"] = loadFile(userIndexHTMLPath)

	// /_/index
	req_path := strings.TrimLeft(strings.Trim(wr.req.URL.Path, "/"), "_")
	if req_path == "" {
		http.Redirect(wr.rw, wr.req, "/_/index", http.StatusFound)
		return
	}
	switch req_path {
	case "/index":
		wr.values["Title"] = "API List"
		wr.apiList()
		return
	case "/api":
		wr.values["Title"] = "Edit"
		wr.apiEdit()
		return
	case "/services":
		wr.values["Title"] = "Service List"
		wr.serviceList()
		return
	case "/about":
		wr.values["Title"] = "About"
		wr.render("about.html", true)
		return
	case "/pref":
		wr.apiPref()
		return
	case "/apipv":
		wr.apiPv()
		return
	case "/login":
		wr.values["Title"] = "Login"
		wr.login()
		return
	case "/logout":
		wr.logout()
		return
	case "/analysis":
		wr.values["Title"] = "Analysis"
		wr.apiAnalysis()
		return
	case "/oauth2_callback":
		wr.oauth2CallBack()
		return
	case "/server_add":
		wr.serverAdd()
		return
	case "/vhost":
		wr.vhostInfo()
		return
	}

	// 	wr.saveSession()
	wr.render("index.html", true)
}

func (wr *webReq) getServerVhostConf() *mainConf {
	return wr.web.apiServer.manager.mainConf
}

func (wr *webReq) saveSession() {
	wr.session.Save(wr.req, wr.rw)
}

func (wr *webReq) oauth2CallBack() {
	oauthconf := wr.getServerVhostConf().Oauth2Conf
	if oauthconf == nil || !oauthconf.Enable {
		wr.alert("oauth login is disabled")
		return
	}
	tok, err := oauthconf.getAccessToken(wr.req)
	if err != nil {
		wr.showError("oauth2 get access_token failed:" + err.Error())
		return
	}

	// 	wr.session.Values["token"] = JSONEncode(tok)

	user, err := oauthconf.getUserInfo(tok)
	if err != nil {
		wr.showError("oauth2 get user_info failed:" + err.Error())
		return
	}

	wr.user = user
	wr.session.Values["user"] = user
	wr.saveSession()

	http.Redirect(wr.rw, wr.req, "/_/index", http.StatusFound)
}

func (wr *webReq) getUser() {
	if u, has := wr.session.Values["user"]; has {
		wr.user = u.(*User)
		log.Println("get user from useesion:", wr.user)
	}
}

func (wr *webReq) getUserID() string {
	if wr.user != nil {
		return wr.user.ID
	}
	return ""
}

func (wr *webReq) apiList() {
	wr.values["apis"] = wr.web.apiServer.Apis
	wr.render("list.html", true)
}

func (wr *webReq) logout() {
	wr.session.Options.MaxAge = -1
	wr.session.Values = make(map[any]any)
	wr.saveSession()
	http.Redirect(wr.rw, wr.req, "/_/index", http.StatusFound)
}

func (wr *webReq) login() {
	oauthconf := wr.getServerVhostConf().Oauth2Conf
	if oauthconf != nil && oauthconf.Enable {
		urlStr := oauthconf.getOauthUrl("http://" + wr.req.Host + "/_/oauth2_callback")
		log.Println("redirect to:", urlStr)
		http.Redirect(wr.rw, wr.req, urlStr, http.StatusFound)
		return
	}
	if wr.req.Method == "POST" {
		id := wr.req.PostFormValue("id")
		psw := wr.req.PostFormValue("psw")
		user := wr.web.userConf.checkUser(id, psw)
		if user == nil {
			log.Println("[warning]login failed;user:", id)
			wr.alert("Login Failed")
			return
		}
		wr.session.Values["user"] = user
		wr.saveSession()
		wr.rw.Write([]byte("<script>parent.location.href='/_/index'</script>"))
	} else {
		wr.render("login.html", true)
	}
}

func (wr *webReq) apiPref() {
	apiID := strings.TrimSpace(wr.req.FormValue("api_id"))
	prefHost := strings.TrimSpace(wr.req.FormValue("host"))

	if apiID == "" {
		wr.json(400, "param empty", nil)
		return
	}

	cookieName := apiCookieName(apiID)

	if prefHost != "" {
		api := wr.web.apiServer.getAPIByID(apiID)
		if api == nil {
			wr.json(400, "Api Not Exists", nil)
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

	wr.json(0, "Success", prefHost)
}

func (wr *webReq) apiPv() {
	apiID := strings.TrimSpace(wr.req.FormValue("api_id"))
	if apiID == "" {
		wr.json(400, "param empty", nil)
		return
	}
	api := wr.web.apiServer.getAPIByID(apiID)
	if api == nil {
		wr.json(400, "Api Not Exists", nil)
		return
	}
	wr.json(0, "Success", api.GetPv())
}

func (wr *webReq) apiAnalysis() {
	name := wr.req.FormValue("id")

	uri := wr.req.FormValue("uri")
	wr.values["uri_prex"] = uri

	if name == "" && uri != "" {
		name = wr.web.apiServer.getAPINameByUri(uri)
	}
	apiName := strings.TrimSpace(name)

	wr.values["Title"] = apiName + " -Analysis"
	if apiName == "" {
		wr.showError("param empty")
		return
	}
	api := wr.web.apiServer.getAPIByID(apiName)
	if api == nil {
		wr.values["error"] = "api not exists!  <a href='/_/api'>add new</a>"
		wr.render("error.html", true)
		return
	}
	wr.values["api"] = api

	// 查看远程存储的地址
	store_view_url := ""
	if wr.web.apiServer.needStore() {
		store_view_url = strings.ReplaceAll(wr.web.apiServer.manager.mainConf.StoreViewUrl, "{host_id}", wr.web.apiServer.ServerVhostConf.Id)
		store_view_url = strings.ReplaceAll(store_view_url, "{api_id}", api.ID)
	}
	wr.values["store_view_url"] = store_view_url

	wr.render("analysis.html", true)
}

func (wr *webReq) showError(msg string) {
	wr.values["error"] = msg
	wr.render("error.html", true)
}

func (wr *webReq) alert(msg string) {
	fmt.Fprintf(wr.rw, `<script>alert("%s")</script>`, StrQuote(msg))
}

func (wr *webReq) alertAndGo(msg string, urlstr string) {
	fmt.Fprintf(wr.rw, `<script>alert("%s");top.location.href="%s";</script>`, StrQuote(msg), urlstr)
}

// JSONResult json result when ajax call
type JSONResult struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (wr *webReq) json(code int, msg string, data any) {
	ret := &JSONResult{Code: code, Msg: msg, Data: data}
	bs, _ := json.Marshal(ret)
	wr.rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	wr.rw.Write(bs)
}

func (wr *webReq) render(tplName string, layout bool) {
	html := renderHTML(tplName, wr.values, true)
	wr.rw.Header().Set("Content-Type", "text/html;charset=utf-8")
	wr.rw.Write([]byte(html))
}

var webTmpName = "tpl_api_front"

func (wr *webReq) serviceList() {
	vhosts := make(map[string][]*serverVhost)
	for _, vhost := range wr.web.apiServer.manager.mainConf.VhostConfs {
		group := vhost.Group
		if group == "" {
			group = "default"
		}
		if _, has := vhosts[group]; !has {
			vhosts[group] = make([]*serverVhost, 0)
		}
		vhosts[group] = append(vhosts[group], vhost)
	}
	wr.values["vhosts"] = vhosts

	wr.render("services.html", true)
}
