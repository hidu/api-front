package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const timeFormatStd string = "2006-01-02 15:04:05"
const timeFormatInt string = "20060102150405"

// SetInterval interval run func
func SetInterval(call func(), sec int64) *time.Ticker {
	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				call()
			}
		}
	}()
	return ticker
}

// FileExists check file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

// DirCheck create dir if not exists
func DirCheck(filePath string) {
	dir := filepath.Dir(filePath)
	if !FileExists(dir) {
		err := os.MkdirAll(dir, 0777)
		log.Println("mkdir dir:", dir, err)
	}
}

// InStringSlice check string in slice
func InStringSlice(str string, strSli []string) bool {
	for _, v := range strSli {
		if str == v {
			return true
		}
	}
	return false
}

// InIntSlice check int in slice
func InIntSlice(id int, idSli []int) bool {
	for _, v := range idSli {
		if id == v {
			return true
		}
	}
	return false
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// StrSliceRandItem get random item  from slice
func StrSliceRandItem(strsli []string) string {
	if len(strsli) == 0 {
		return ""
	}
	n := time.Now().UnixNano() % int64(len(strsli))
	return strsli[n]
}

// StrSliceIntersectGetOne  get one of a ,which a is in b
func StrSliceIntersectGetOne(a, b []string) string {
	c := make([]string, 0, len(b))
	for _, v := range a {
		if InStringSlice(v, b) {
			c = append(c, v)
		}
	}
	return StrSliceRandItem(c)
}

// URLPathClean clean url path
func URLPathClean(urlPath string) string {
	flag := strings.HasSuffix(urlPath, "/")
	str := path.Clean(fmt.Sprintf("/%s", urlPath))
	if flag && str != "/" {
		return fmt.Sprintf("%s/", str)
	}
	return str
}

var textContentTypes = []string{"text", "javascript", "json"}

// IsContentTypeText check contentType is text
func IsContentTypeText(contentType string) bool {
	for _, v := range textContentTypes {
		if strings.Contains(contentType, v) {
			return true
		}
	}
	return false
}

// LoadJSONFile easy load json file
func LoadJSONFile(jsonPath string, obj interface{}) error {
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &obj)
	return err
}

// IsRequestDumpBody if not big body,can  dump request
func IsRequestDumpBody(req *http.Request) bool {
	switch req.Method {
	case "GET":
	case "DELETE":
	case "HEAD":
		return true
	}
	if IsContentTypeText(req.Header.Get("Content-Type")) {
		return true
	}

	if req.ContentLength > 0 && req.ContentLength < 1e7 {
		return true
	}

	return false
}

func forgetRead(reader *io.ReadCloser) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	io.Copy(buf, *reader)
	*reader = ioutil.NopCloser(buf).(io.ReadCloser)
	return bytes.NewBuffer(buf.Bytes())
}

// ReqCookieHidden hidden request cookie value
func ReqCookieHidden(str string) string {
	cs := []string{}
	arr := strings.Split(str[7:], "; ")
	for _, v := range arr {
		tmp := strings.SplitN(v, "=", 2)
		cs = append(cs, fmt.Sprintf("%s=hidden", tmp[0]))
	}
	return str[:7] + strings.Join(cs, "; ") + "\r\n"
}

// ResCookieSetHidden hidden response cookie value
func ResCookieSetHidden(str string) string {
	arr := strings.SplitN(strings.TrimSpace(str[11:]), ";", 2)
	tmp := strings.SplitN(arr[0], "=", 2)
	setCookie := str[:11] + tmp[0] + "=hidden"
	if len(arr) > 1 {
		setCookie += ";" + arr[1]
	}
	return setCookie + "\r\n"
}

func gzipDocode(buf *bytes.Buffer) string {
	if buf.Len() < 1 {
		return ""
	}
	gr, err := gzip.NewReader(buf)
	defer gr.Close()
	if err == nil {
		bdBt, _ := ioutil.ReadAll(gr)
		return string(bdBt)
	}
	log.Println("unzip body failed", err)
	return ""
}

func loadFile(file string) string {
	if !FileExists(file) {
		return ""
	}
	ds, _ := ioutil.ReadFile(file)
	return string(ds)
}

func JSONEncode(obj interface{}) string {
	bs, _ := json.Marshal(obj)
	return string(bs)
}

func StrQuote(str string) string {
	return strings.Replace(str, `"`, "\\\"", -1)
}
