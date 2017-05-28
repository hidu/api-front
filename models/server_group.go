package models

import (
	"errors"
	"github.com/astaxie/beego/orm"
	"time"
)

type ServerGroup struct {
	ID     int64 `orm:"pk;auto;column(group_id)"`
	Status int   `orm:"column(status);default(1)"`

	Name     string `orm:"size(64);column(name)"`
	Intro    string `orm:"size(10000);column(intro)"`
	HomePage string `orm:"size(1024);column(home_page)"`

	CreateTime  time.Time     `orm:"auto_now_add;type(datetime);column(ctime)"`
	UpdateTime  time.Time     `orm:"auto_now;type(datetime);column(mtime)"`
	ServerNodes []*ServerNode `orm:"-"`
}

func (g *ServerGroup) TableName() string {
	return "server_group"
}

func (g *ServerGroup) Create() (id int64, err error) {
	o := orm.NewOrm()
	return o.Insert(g)
}

func (g *ServerGroup) Read() error {
	o := orm.NewOrm()
	err := o.Read(g)
	g.GetServerNodes()
	return err
}

func (g *ServerGroup) Delete() (num int64, err error) {
	o := orm.NewOrm()
	return o.Delete(g)
}

func (g *ServerGroup) Update() error {
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

func (g *ServerGroup) GetServerNodes() []*ServerNode {
	cond := orm.NewCondition().And("group_id", g.ID)
	ls := ListAllServerNode(cond)
	g.ServerNodes = ls
	return ls
}

func (g *ServerGroup) Query() orm.QuerySeter {
	return orm.NewOrm().QueryTable(g)
}

func ListAllServerGroup() []*ServerGroup {
	var ls []*ServerGroup
	new(ServerGroup).Query().All(&ls)
	return ls
}
