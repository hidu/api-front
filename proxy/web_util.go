package proxy

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"text/template"
	"time"

	"github.com/hidu/goutils/html_util"
)

func readerHTMLInclude(fileName string) string {
	html := string(Asset.GetContent("/resource/tpl/" + fileName))
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

func renderHTML(fileName string, values map[string]any, layout bool) string {
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
		"str_eq": func(x, y any) bool {
			ret := fmt.Sprintf("%x", x) == fmt.Sprintf("%x", y)
			return ret
		},
		"my_include": func(fileName string) string {
			return "include (" + fileName + ") with Delims {%my_include %}"
		},
		"my_include_html": func(fileName string) string {
			return string(Asset.GetContent("/resource/tpl/" + fileName))
		},
		"h": func(str string) string {
			return html.EscapeString(str)
		},
		"join": func(arr []string) string {
			return strings.Join(arr, "\n")
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
	// 	return body
	return html_util.Html_reduceSpace(body)
}
