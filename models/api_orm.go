package models

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func ORMInit() {
	log.Println("orm_init_start")
	
	orm.RegisterModel(
		new(ServerGroup),
		new(ServerNode),
		new(Location),
		new(Backend),
	)
	orm.Debug=true
	
	orm.RegisterDriver("sqlite", orm.DRSqlite)
	orm.RegisterDriver("mysql", orm.DRMySQL)
	db_driver := beego.AppConfig.DefaultString("db_driver", "sqlite3")
	db_dsn := beego.AppConfig.String("db_dsn")
	orm.RegisterDataBase("default", db_driver, db_dsn)
	
	log.Println("orm.RunSyncdb_start")
	orm.RunSyncdb("default", false, true)
	log.Println("orm.RunSyncdb_finish")
}
