package proxy

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"
	"github.com/hidu/goutils"
)

type WebAdmin struct {
	mimoServer *MimoServer
}

func NewWebAdmin(mimo *MimoServer) *WebAdmin {
	ser := &WebAdmin{
		mimoServer: mimo,
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
	wr.values["version"] = 0.1
	wr.values["base_url"] = "http://" + wr.req.Host + "/"
	wr.values["server_list"] = wr.web.mimoServer.manager.ServerConf.Server
	host_info := strings.Split(wr.req.Host, ":")
	if host_info[1] == "" {
		host_info[1] = "80"
	}
	wr.values["host_name"] = host_info[0]
	port, _ := strconv.ParseInt(host_info[1], 10, 64)
	wr.values["host_port"] = int(port)
	wr.values["conf"] = wr.web.mimoServer.ServerConf

	if wr.req.URL.Path == "/_api" {
		wr.apiEdit()
		return
	}
	if wr.req.URL.Path == "/_apis" {
		wr.apisList()
		return
	}
	wr.render("index.html", true)
}

func (wr *webReq) apisList() {
	apiNames := make([]string, 0, 100)

	for name := range wr.web.mimoServer.Apis {
		apiNames = append(apiNames, name)
	}
	wr.values["apiNames"] = apiNames
	wr.render("list.html", true)
}

func (wr *webReq) alert(msg string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s")</script>`, msg)))
}
func (wr *webReq) alertAndGo(msg string, urlstr string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s");top.location.href="%s";</script>`, msg, urlstr)))
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
			apiOld := wr.web.mimoServer.getApiByName(name)
			if apiOld == nil {
				wr.values["error"] = "api not exists!  <a href='/_api'>add new</a>"
				wr.render("error.html", true)
				return
			}
			api = apiOld.Clone()
		} else {
			api = NewApi()
		}
		api.Hosts.AddNewHost(NewHost(web_tmp_name, "http://127.0.0.1/", false))
		if len(api.Paths) == 0 {
			api.Paths.RegirestNewPath(NewApiPath("/", "default"))
		} else {
			api.Paths.RegirestNewPath(NewApiPath("/your_path_change", "your_path_change"))
		}
		wr.values["api"] = api
		wr.values["api_url"] = "http://" + req.Host + "/" + api.Name
		wr.render("api.html", true)
		return
	}

	do := req.FormValue("do")
	switch do {
	case "base":
		wr.apiBaseSave()
	case "path":
		wr.apiPathSave()

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
	api := wr.web.mimoServer.getApiByName(apiName)
	api_name_orig := req.FormValue("api_name_orig")
	if api_name_orig == "" && api != nil {
		wr.alert("失败：新创建的模块已经存在")
		return
	}
	if api == nil {
		api = wr.web.mimoServer.newApi(apiName)
	}
	host_name := req.PostForm["host_name"]
	host_name_orig := req.PostForm["host_name_orig"]
	host_url := req.PostForm["host_url"]
	host_note := req.PostForm["host_note"]

	tmp := make(map[string]string)
	for _, val := range host_name {
		if _, has := tmp[val]; has {
			wr.alert("别名:" + val + "，重复了")
			return
		}
	}

	for i, name := range host_name {
		if name == "" || name == web_tmp_name {
			continue
		}
		host := NewHost(name, host_url[i], true)
		host.Note = host_note[i]

		//		wr.web.mimoServer.
		api.Hosts.AddNewHost(host)
		name_orig := host_name_orig[i]
		api.HostRename(name_orig, name)
	}
	api.HostCheckDelete(host_name)

	if len(host_name) != len(host_url) || len(host_name) != len(host_note) {
		wr.alert("保存失败：后端服务数据错误")
	}

	if api == nil {
		api = wr.web.mimoServer.newApi(apiName)
		indexPath := NewApiPath("/", "default all")
		api.Paths.RegirestNewPath(indexPath)
	}

	api.Note = req.FormValue("note")
	api.TimeoutMs = int(timeout)

	if api_name_orig != apiName {
		wr.web.mimoServer.deleteApi(api_name_orig)
	}

	err = api.Save()
	if err != nil {
		wr.alert("保存失败：" + err.Error())
		return
	}
	wr.web.mimoServer.loadApi(apiName)
	wr.alertAndGo("已经更新！", "/_api?name="+apiName)
}

func (wr *webReq) apiPathSave() {
	req := wr.req
	apiName := req.FormValue("api_name")
	api := wr.web.mimoServer.getApiByName(apiName)
	if api == nil {
		wr.alert("api模块不存在")
		return
	}

	backend_path := strings.TrimSpace(req.FormValue("path"))
	if !api.IsValidPath(backend_path) {
		wr.alert("api绑定路径错误,正确的格式：/ 或者 /a")
		return
	}
	path_orig := strings.TrimSpace(req.FormValue("path_orig"))

	note := req.FormValue("note")

	if backend_path != path_orig && api.isPathRegistered(backend_path) {
		wr.alert(backend_path + " 已经存在")
		return
	}
	host_names := req.Form["host_names"]
	if len(host_names) == 0 {
		host_names = []string{}
	}
	api_path := NewApiPath(backend_path, note)
	api_path.HostNames = host_names

	api.Paths.RegirestNewPath(api_path)
	api.Paths.PathRename(path_orig, backend_path)

	err := api.Save()
	if err != nil {
		wr.alert("保存配置失败:" + err.Error())
		return
	}
	wr.web.mimoServer.loadApi(apiName)
	wr.alert("已经更新！")
}

func render_html(fileName string, values map[string]interface{}, layout bool) string {

	html := Assest.GetContent("/res/tpl/" + fileName)
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
			//			fmt.Println("str_eq:",x,y,ret)
			return ret
		},
	}

	tpl, _ := template.New("page").Funcs(myfn).Parse(string(html))

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
