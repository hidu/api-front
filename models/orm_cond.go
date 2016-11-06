package models

import (
	"github.com/astaxie/beego/orm"
)

type OrmCond struct {
	Filter map[string][]interface{}
}

func (c *OrmCond) AddFilter(name string, vals ...interface{}) {
	c.Filter[name] = vals
}

func (c *OrmCond) BuildQuery(query orm.QuerySeter) {
	for name, vals := range c.Filter {
		query.Filter(name, vals...)
	}
}

func NewOrmCond() *OrmCond {
	return &OrmCond{
		Filter: make(map[string][]interface{}),
	}
}
