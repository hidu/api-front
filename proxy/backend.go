package proxy

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

type Backend struct {
	Url    string `json:"url"`
	Master bool   `json:"master"`
	Note   string `json:"note"`
	Weight int    `json:"-"`
}

func (back *Backend) init() error {
	u, err := url.Parse(back.Url)
	eMsg := "backend url [" + back.Url + "] "
	if err != nil {
		log.Println(eMsg+" parse failed:", err)
		return err
	}
	if u.Scheme != "http" {
		log.Println(eMsg + " ,schema is not http")
		return fmt.Errorf("schema is not http")
	}
	if back.Master {
		back.Weight = 1
	}
	return nil
}

func NewBackend(urlStr string, note string, master bool) (*Backend, error) {
	back := &Backend{
		Url:    strings.TrimSpace(urlStr),
		Master: master,
		Note:   note,
	}
	err := back.init()
	return back, err
}

type Backends []*Backend

func (backs *Backends) init() error {
	masterTotal := 0
	for _, bak := range *backs {
		err := bak.init()
		if err != nil {
			continue
		}
		if bak.Master {
			masterTotal++
		}
		log.Println("backend url:", bak.Url, "isMaster:", bak.Master)
	}

	if masterTotal == 0 {
		for _, bak := range *backs {
			bak.Weight = 1
		}
	}
	return nil
}

func (backs *Backends) GetMasterIndex() int {
	masterTotal := 0
	for _, bak := range *backs {
		if bak.Weight > 0 {
			masterTotal++
		}
	}
	if masterTotal < 1 {
		return 0
	}

	indexM := randR.Int() % masterTotal
	i := 0
	for n, bak := range *backs {
		if bak.Weight > 0 && i >= indexM {
			i++
			return n
		}
	}
	return 0

}
