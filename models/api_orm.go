package models

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func ORMInit() {

	orm.RegisterModel(
		new(ServerGroup),
		new(ServerHost),
		new(Location),
		new(Backend),
	)
	orm.RegisterDriver("sqlite", orm.DRSqlite)
	orm.RegisterDriver("mysql", orm.DRMySQL)
	db_driver := beego.AppConfig.DefaultString("db_driver", "sqlite3")
	db_dsn := beego.AppConfig.String("db_dsn")
	orm.RegisterDataBase("default", db_driver, db_dsn)
	orm.RunSyncdb("default", false, true)
}
