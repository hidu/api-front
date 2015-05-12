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

	if req.URL.Path == "/_module" {
		web.moduleEdit(rw, req)
		return
	}

	values := make(map[string]interface{})
	moduleNames := make([]string, 0, 100)

	for name := range web.mimoServer.Modules {
		moduleNames = append(moduleNames, name)
	}
	values["moduleNames"] = moduleNames

	html := render_html("index.html", values, true)
	rw.Write([]byte(html))
}

func (web *WebAdmin) moduleEdit(rw http.ResponseWriter, req *http.Request) {
	values := make(map[string]interface{})
	values["base_url"] = "http://" + req.Host + "/"
	name := req.FormValue("name")
	if req.Method != "POST" {
		tpl := "module.html"
		var mod *Module
		if name != "" {
			mod = web.mimoServer.getModuleByName(name)
			if mod == nil {
				tpl = "error.html"
				values["error"] = "module not exists!  <a href='/_module'>add new</a>"
			}
		} else {
			mod = NewModule()
		}
		if len(mod.Paths) == 0 {
			mod.Paths["/"] = make(Backends, 0)
		} else {
			mod.Paths["请修改,如: /"] = make(Backends, 0)
		}
		values["module"] = mod

		html := render_html(tpl, values, true)
		rw.Write([]byte(html))
	} else {
		do := req.FormValue("do")
		switch do {
		case "base":
			web.moduleBaseSave(rw, req)
		case "path":
			web.moduleBackendSave(rw, req)

		}
	}
}
func (web *WebAdmin) moduleBaseSave(rw http.ResponseWriter, req *http.Request) {
	timeout, err := strconv.ParseInt(req.FormValue("timeout"), 10, 64)
	if err != nil {
		rw.Write([]byte(`<script>alert("超时时间错误,不是int")</script>`))
		return
	}

	moduleName := req.FormValue("module_name")
	mod := web.mimoServer.getModuleByName(moduleName)
	module_name_orig := req.FormValue("module_name_orig")
	if module_name_orig == "" && mod != nil {
		rw.Write([]byte(`<script>alert("失败：新创建的模块已经存在")</script>`))
		return
	}
	if mod == nil {
		mod = web.mimoServer.newModule(moduleName)
	}

	mod.Note = req.FormValue("note")
	mod.TimeoutMs = int(timeout)

	err = mod.Save()
	if err != nil {
		rw.Write([]byte(fmt.Sprintf(`<script>alert("保存失败：%s")</script>`, err.Error())))
		return
	}
	//web.mimoServer.loadModule(moduleName)
	rw.Write([]byte(`<script>alert("已经更新！")</script>`))
}

func (web *WebAdmin) moduleBackendSave(rw http.ResponseWriter, req *http.Request) {
	moduleName := req.FormValue("module_name")
	mod := web.mimoServer.getModuleByName(moduleName)
	if mod == nil {
		rw.Write([]byte(`<script>alert("模块不存在")</script>`))
		return
	}

	backend_path := strings.TrimSpace(req.FormValue("path"))
	if backend_path == "" {
		rw.Write([]byte(`<script>alert("绑定路径不能为空")</script>`))
		return
	}
	urls := req.Form["url"]
	notes := req.Form["note"]
	masters := req.Form["master"]
	if len(urls) != len(notes) || len(urls) != len(masters) {
		rw.Write([]byte(`<script>alert("数据不完整")</script>`))
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
			rw.Write([]byte(fmt.Sprintf(`<script>alert("[%s]解析出错:%s")</script>`, urlStr, err.Error())))
			return
		}
		backends = append(backends, back)
	}

	mod.UpdateBackends(backend_path, backends)
	err := mod.Save()
	if err != nil {
		rw.Write([]byte(fmt.Sprintf(`<script>alert("保存配置失败：%s")</script>`, err.Error())))
		return
	}
	//web.mimoServer.loadModule(moduleName)
	rw.Write([]byte(`<script>alert("已经更新！")</script>`))
}

func render_html(fileName string, values map[string]interface{}, layout bool) string {
	values["version"] = 0.1

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
