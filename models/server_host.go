package models

import (
	"errors"
	"github.com/astaxie/beego/orm"
	"time"
)

type ServerHost struct {
	ID      int64  `orm:"pk;auto;column(id)"`
	Status  int    `orm:"column(status)"`
	GroupID int64  `orm:"column(api_group_id);index"`
	Name    string `orm:"size(64);column(name)"`
	Intro   string `orm:"size(10000);column(intro)"`
	Port    int    `orm:"column(port);unique"`
	UniqKey string `orm:"size(32);unique;column(uniq_key)"`

	CreateTime time.Time `orm:"auto_now_add;type(datetime);column(ctime)"`
	UpdateTime time.Time `orm:"auto_now;type(datetime);column(mtime)"`

	Group     *ServerGroup `orm:"-"`
	Locations []*Location  `orm:"-"`
}

func (s *ServerHost) TableName() string {
	return "api_server_host"
}

func (g *ServerHost) CheckGroupExists() error {
	if g.GroupID < 1 {
		return errors.New("empty group_id")
	}
	group := new(ServerGroup)
	group.ID = g.GroupID
	err := group.Read()
	if err != nil {
		return err
	}
	return nil
}

func (g *ServerHost) Create() (id int64, err error) {

	if err := g.CheckGroupExists(); err != nil {
		return 0, err
	}

	o := orm.NewOrm()
	return o.Insert(g)
}

func (g *ServerHost) Read() error {
	o := orm.NewOrm()
	err := o.Read(g)
	//	g.GetLocations()
	return err
}

func (g *ServerHost) Delete() (num int64, err error) {
	o := orm.NewOrm()
	return o.Delete(g)
}

func (g *ServerHost) Update() error {
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

func (g *ServerHost) GetLocations() []*Location {
	cond := NewOrmCond()
	cond.AddFilter("api_server_id", g.ID)
	ls := ListAllLocation(cond)
	g.Locations = ls
	return ls
}

func (g *ServerHost) Query() orm.QuerySeter {
	return orm.NewOrm().QueryTable(g)
}

func ListAllServerHost(cond *orm.Condition) []*ServerHost {
	var ls []*ServerHost
	query := new(ServerHost).Query().SetCond(cond)
	query.All(&ls)
	return ls
}
