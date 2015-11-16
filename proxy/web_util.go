package proxy

import (
	"bytes"
	"fmt"
	"github.com/hidu/goutils"
	"html"
	"text/template"
	"time"
)

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
	//	return body
	return utils.Html_reduceSpace(body)
}
