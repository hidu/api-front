package mimo

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
)

type Module struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Enable    bool       `json:"enable"`
	Backends  []*Backend `json:"backends"`
	HasMaster bool
}

func loadModules(conf_dir string) []*Module {
	fileNames, _ := filepath.Glob(conf_dir + "/*.json")
	mods := make([]*Module, 0)
	for _, fileName := range fileNames {
		mod, err := loadOneModule(fileName)
		relName, _ := filepath.Rel(filepath.Dir(conf_dir), fileName)
		if err != nil {
			log.Println("load module [", relName, "] failed", err)
			continue
		}
		log.Println("load module [", relName, "] success")
		mods = append(mods, mod)
	}
	return mods
}

func loadOneModule(conf_path string) (*Module, error) {
	data, err := ioutil.ReadFile(conf_path)
	if err != nil {
		return nil, err
	}
	var mod *Module
	err = json.Unmarshal(data, &mod)
	if err != nil {
		return nil, err
	}
	return mod, nil
}
