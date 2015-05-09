package mimo

type Module struct {
	Name   string
	Desc   string              `json:"desc"`
	Enable bool                `json:"enable"`
	Paths  map[string]Backends `json:"paths"`
}

type Backends []*Backend
