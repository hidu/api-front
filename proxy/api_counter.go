package proxy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

type Counter struct {
	Pv        map[string]uint64 `json:"pv"`
	rw        sync.RWMutex
	TotalPv   uint64 `json:"total"`
	filePath  string
	lastMod   time.Time
	lastWrite time.Time
}

func NewCounter(apiServer *ApiServer) *Counter {
	jsonPath := apiServer.GetConfDir() + "/_counter.json"
	var counter *Counter
	err := LoadJsonFile(jsonPath, &counter)
	if err != nil {
		log.Println("load counter failed:", jsonPath, ",err:", err)
		counter = new(Counter)
	}
	counter.filePath = jsonPath
	if counter.Pv == nil {
		counter.Pv = make(map[string]uint64)
	}
	counter.lastMod = time.Now()
	counter.lastWrite = counter.lastMod
	go counter.AutoSave(60)

	return counter
}

func (c *Counter) PvInc(name string) uint64 {
	c.rw.Lock()
	defer c.rw.Unlock()
	if _, has := c.Pv[name]; !has {
		c.Pv[name] = 0
	}
	c.Pv[name]++
	c.TotalPv++
	c.lastMod = time.Now()
	return c.TotalPv
}

func (c *Counter) GetPv(name string) uint64 {
	c.rw.RLock()
	defer c.rw.RUnlock()
	if v, has := c.Pv[name]; has {
		return v
	}
	return 0
}

func (c *Counter) GetTotalPv() uint64 {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.TotalPv
}

func (c *Counter) AutoSave(sec int64) {
	t := time.NewTicker(time.Duration(sec) * time.Second)
	for {
		select {
		case <-t.C:
			if c.lastWrite.UnixNano() < c.lastMod.UnixNano() {
				c.SaveFile()
			}
		}
	}
}

func (c *Counter) SaveFile() error {
	log.Println("save counter file:", c.filePath)
	c.rw.RLock()
	defer c.rw.RUnlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	c.lastWrite = time.Now()
	return ioutil.WriteFile(c.filePath, data, 0666)
}
