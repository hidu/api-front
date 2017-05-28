package models

import (
	"errors"
	"github.com/astaxie/beego/orm"
	"time"
)

type ServerNode struct {
	ID      int64  `orm:"pk;auto;column(node_id)"`
	GroupID int64  `orm:"column(group_id);index"`
	
	Status  int    `orm:"column(status)"`
	Name    string `orm:"size(64);column(name)"`
	Intro   string `orm:"size(10000);column(intro)"`
	Port    int    `orm:"column(port);unique"`
	UniqKey string `orm:"size(32);unique;column(uniq_key)"`

	CreateTime time.Time `orm:"auto_now_add;type(datetime);column(ctime)"`
	UpdateTime time.Time `orm:"auto_now;type(datetime);column(mtime)"`

	Group     *ServerGroup `orm:"-"`
	Locations []*Location  `orm:"-"`
}

func (s *ServerNode) TableName() string {
	return "server_node"
}

func (g *ServerNode) CheckGroupExists() error {
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

func (g *ServerNode) Create() (id int64, err error) {

	if err := g.CheckGroupExists(); err != nil {
		return 0, err
	}

	o := orm.NewOrm()
	return o.Insert(g)
}

func (g *ServerNode) Read() error {
	o := orm.NewOrm()
	err := o.Read(g)
	//	g.GetLocations()
	return err
}

func (g *ServerNode) Delete() (num int64, err error) {
	o := orm.NewOrm()
	return o.Delete(g)
}

func (g *ServerNode) Update() error {
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

func (g *ServerNode) GetLocations() []*Location {
	cond := orm.NewCondition().And("node_id", g.ID)
	ls := ListAllLocation(cond)
	g.Locations = ls
	return ls
}

func (g *ServerNode) Query() orm.QuerySeter {
	return orm.NewOrm().QueryTable(g)
}

func ListAllServerNode(cond *orm.Condition) []*ServerNode {
	var ls []*ServerNode
	query := new(ServerNode).Query().SetCond(cond)
	query.All(&ls)
	return ls
}
