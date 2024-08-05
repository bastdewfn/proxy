package database

import "github.com/astaxie/beego"

type Database struct {
	Driver          string
	Source          string
	ConnMaxIdleTime int
	ConnMaxLifeTime int
	MaxIdleConns    int
	MaxOpenConns    int
	Registers       []DBResolverConfig
}

type DBResolverConfig struct {
	Sources  []string
	Replicas []string
	Policy   string
	Tables   []string
}

var (
	DatabaseConfig  = new(Database)
	DatabasesConfig = make(map[string]*Database)
)

func (database *Database) InitDbConfig() {
	DatabaseConfig.Driver = beego.AppConfig.String("db_driver")
	DatabaseConfig.Source = beego.AppConfig.String("db_source")
	DatabaseConfig.ConnMaxIdleTime, _ = beego.AppConfig.Int("db_ConnMaxIdleTime")
	DatabaseConfig.ConnMaxLifeTime, _ = beego.AppConfig.Int("db_ConnMaxLifeTime")
	DatabaseConfig.MaxIdleConns, _ = beego.AppConfig.Int("db_maxIdleConns")
	DatabaseConfig.MaxOpenConns, _ = beego.AppConfig.Int("db_maxOpenConns")
	DatabasesConfig["*"] = DatabaseConfig
}
