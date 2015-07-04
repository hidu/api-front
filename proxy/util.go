package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const TIME_FORMAT_STD string = "2006-01-02 15:04:05"
const TIME_FORMAT_INT string = "20060102150405"

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

func File_exists(file_path string) bool {
	_, err := os.Stat(file_path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func DirCheck(file_path string) {
	dir := filepath.Dir(file_path)
	if !File_exists(dir) {
		err := os.MkdirAll(dir, 0777)
		log.Println("mkdir dir:", dir, err)
	}
}

func In_StringSlice(str string, strSli []string) bool {
	for _, v := range strSli {
		if str == v {
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

func StrSliceRandItem(strsli []string) string {
	if len(strsli) == 0 {
		return ""
	}
	n := time.Now().UnixNano() % int64(len(strsli))
	return strsli[n]
}

func StrSliceIntersectGetOne(a, b []string) string {
	c := make([]string, 0, len(b))
	for _, v := range a {
		if In_StringSlice(v, b) {
			c = append(c, v)
		}
	}
	return StrSliceRandItem(c)
}

func UrlPathClean(urlPath string) string {
	str := path.Clean(fmt.Sprintf("/%s/", urlPath))
	if strings.HasSuffix(str, "/") {
		return str
	} else {
		return fmt.Sprintf("%s/", str)
	}
}
