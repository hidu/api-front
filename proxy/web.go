package proxy

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"
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
	req := wr.req
	wr.values["version"] = 0.1
	wr.values["base_url"] = "http://" + req.Host + "/"
	if wr.req.URL.Path == "/_api" {
		wr.apiModuleEdit()
		return
	}
	if wr.req.URL.Path == "/_apis" {
		wr.apisList()
		return
	}
	wr.render("index.html", true)
}

func (wr *webReq) apisList() {
	moduleNames := make([]string, 0, 100)

	for name := range wr.web.mimoServer.Modules {
		moduleNames = append(moduleNames, name)
	}
	wr.values["moduleNames"] = moduleNames
	wr.render("list.html", true)
}

func (wr *webReq) alert(msg string) {
	wr.rw.Write([]byte(fmt.Sprintf(`<script>alert("%s")</script>`, msg)))
}

func (wr *webReq) render(tplName string, layout bool) {
	html := render_html(tplName, wr.values, true)
	wr.rw.Write([]byte(html))
}

func (wr *webReq) apiModuleEdit() {
	req := wr.req
	name := req.FormValue("name")
	if req.Method != "POST" {
		var mod *Module
		if name != "" {
			modOld := wr.web.mimoServer.getModuleByName(name)
			if modOld == nil {
				wr.values["error"] = "module not exists!  <a href='/_module'>add new</a>"
				wr.render("error.html", true)
				return
			}
			mod = modOld.Clone()
		} else {
			mod = NewModule()
		}
		if len(mod.Paths) == 0 {
			mod.Paths["/"] = make(Backends, 0)
		} else {
			mod.Paths["请修改,如: /"] = make(Backends, 0)
		}
		wr.values["module"] = mod
		wr.values["widget_backend"]=render_html("widget_backend.html",wr.values,false)
		wr.values["api_url"]="http://"+req.Host+"/"+mod.Name
		wr.render("api.html", true)
		return
	}

	do := req.FormValue("do")
	switch do {
	case "base":
		wr.moduleBaseSave()
	case "path":
		wr.moduleBackendSave()

	}
}
func (wr *webReq) moduleBaseSave() {
	req := wr.req
	timeout, err := strconv.ParseInt(req.FormValue("timeout"), 10, 64)
	if err != nil {
		wr.alert("超时时间错误,不是int")
		return
	}

	moduleName := req.FormValue("module_name")
	mod := wr.web.mimoServer.getModuleByName(moduleName)
	module_name_orig := req.FormValue("module_name_orig")
	if module_name_orig == "" && mod != nil {
		wr.alert("失败：新创建的模块已经存在")
		return
	}
	if mod == nil {
		mod = wr.web.mimoServer.newModule(moduleName)
	}

	mod.Note = req.FormValue("note")
	mod.TimeoutMs = int(timeout)

	err = mod.Save()
	if err != nil {
		wr.alert("保存失败：" + err.Error())
		return
	}
	wr.web.mimoServer.loadModule(moduleName)
	wr.alert("已经更新！")
}

func (wr *webReq) moduleBackendSave() {
	req := wr.req
	moduleName := req.FormValue("module_name")
	mod := wr.web.mimoServer.getModuleByName(moduleName)
	if mod == nil {
		wr.alert("api模块不存在")
		return
	}

	backend_path := strings.TrimSpace(req.FormValue("path"))
	if !mod.IsValidPath(backend_path) {
		wr.alert("api绑定路径错误,正确的格式：/ 或者 /a")
		return
	}
	path_orig := strings.TrimSpace(req.FormValue("path_orig"))

	urls := req.Form["url"]
	notes := req.Form["note"]
	masters := req.Form["master"]
	if len(urls) != len(notes) || len(urls) != len(masters) {
		wr.alert("数据不完整")
		return
	}
	if backend_path != path_orig && mod.isPathRegistered(backend_path) {
		wr.alert(backend_path + " 已经存在")
		return
	}

	backends := make(Backends, 0, len(urls))
	for i, urlStr := range urls {
		urlStr = strings.TrimSpace(urlStr)
		if urlStr == "" {
			continue
		}
		note := notes[i]
		master := masters[i]
		back, err := NewBackend(urlStr, note, master == "1")
		if err != nil {
			wr.alert(urlStr + " 解析出错:" + err.Error())
			return
		}
		backends = append(backends, back)
	}
	backends.init()
	if len(backends) > 0 {
		mod.UpdateBackends(backend_path, backends)
	} else {
		mod.deletePath(backend_path)
	}
	if backend_path != path_orig {
		mod.deletePath(path_orig)
	}

	err := mod.Save()
	if err != nil {
		wr.alert("保存配置失败:" + err.Error())
		return
	}
	wr.web.mimoServer.loadModule(moduleName)
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
	return body
}
