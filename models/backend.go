package models

import (
	"errors"
	"github.com/astaxie/beego/orm"
	"time"
)

type Backend struct {
	ID     int64  `orm:"pk;auto;column(id)"`
	LocationId int       `orm:"column(location_id);index"`
	
	Status int    `orm:"column(status)"`
	Name   string `orm:"size(64);column(name)"`
	Intro  string `orm:"site(10000);column(intro)"`
	URLStr string `orm:"size(10000);column(url)"`

	CreateTime time.Time `orm:"auto_now_add;type(datetime);column(ctime)"`
	UpdateTime time.Time `orm:"type(datetime);column(mtime)"`

	Location *Location `orm:"-"`
}

func (h *Backend) TableName() string {
	return "api_backend"
}

func (g *Backend) Create() (id int64, err error) {
	o := orm.NewOrm()
	return o.Insert(g)
}

func (g *Backend) Read() error {
	o := orm.NewOrm()
	return o.Read(g)
}

func (g *Backend) Delete() (num int64, err error) {
	o := orm.NewOrm()
	return o.Delete(g)
}

func (g *Backend) Update() error {
	if g.ID < 1 {
		return errors.New("id is required")
	}
	o := orm.NewOrm()
	_, err := o.Update(g)
	if err != nil {
		return err
	}
	return nil
}

func (g *Backend) Query() orm.QuerySeter {
	return orm.NewOrm().QueryTable(g)
}

func ListAllBackend(cond *orm.Condition) []*Backend {
	var ls []*Backend
	query := new(Backend).Query().SetCond(cond)
	query.All(&ls)
	return ls
}
