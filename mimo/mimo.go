package mimo

import (
	"fmt"
	"log"
	"net/http"
)

type MimoServer struct {
	Port    int
	Enable  bool
	Modules []*Module
	ConfDir string
}

func NewMimoServer(port int) *MimoServer {
	mimo := &MimoServer{Port: port}
	return mimo
}

func (mimo *MimoServer) Start() error {
	addr := fmt.Sprintf(":%d", mimo.Port)
	log.Println("start server:", addr)

	mimo.loadModules()

	err := http.ListenAndServe(addr, nil)
	return err
}

func (mimo *MimoServer) loadModules() {
	mimo.Modules = loadModules(fmt.Sprintf("%s/%d", mimo.ConfDir, mimo.Port))
}
