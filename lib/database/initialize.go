package database

import (
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"dewfn.com/nps/lib/common"
	"dewfn.com/nps/lib/crypt"
	"dewfn.com/nps/lib/global"
	"time"
)

// Setup 配置数据库
func Setup() {
	for k := range DatabasesConfig {
		setupSimpleDatabase(k, DatabasesConfig[k])
	}
}

func setupSimpleDatabase(host string, c *Database) {
	if global.Driver == "" {
		global.Driver = c.Driver
	}
	logs.Debug("%s => %s", host, c.Source)
	c.Source = crypt.DecryptDbCon(c.Source)

	newLogger := &common.SqlMsg{
		Config: logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  false,       // 禁用彩色打印
		},
	}

	db, err := gorm.Open(mysql.Open(c.Source), &gorm.Config{
		Logger: newLogger, //打印所有执行的sql语句
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		}})
	if err != nil {
		logs.Error("创建数据库连接失败:%v", err)
	}
	mysql, err := db.DB()
	if err != nil {
		defer mysql.Close()
		logs.Error("创建数据库连接失败:%v", err)
	}
	mysql.SetMaxIdleConns(c.MaxIdleConns)
	mysql.SetMaxOpenConns(c.MaxOpenConns)
	mysql.SetConnMaxIdleTime(time.Duration(c.ConnMaxIdleTime))
	mysql.SetConnMaxLifetime(time.Duration(c.ConnMaxLifeTime))

	global.App.SetDb(host, db)
}
