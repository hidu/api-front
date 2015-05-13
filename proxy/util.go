package proxy

import (
	"log"
	"os"
	"path/filepath"
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
