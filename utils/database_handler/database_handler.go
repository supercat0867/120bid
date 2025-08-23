package database_handler

import (
	"120bid/config"
	"120bid/utils/api_helper"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	// 自定义gorm日志
	newLogger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)
)

// Migrate 表迁移
func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(&api_helper.Data{})
	if err != nil {
		panic(fmt.Sprintf("数据表迁移失败：%s", err.Error()))
	}
}

// NewMySQL 创建数据库连接
func NewMySQL(cfg *config.Config) *gorm.DB {
	dbConf := cfg.Database.MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=True&loc=Local", dbConf.User, dbConf.Password, dbConf.Address, dbConf.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			NoLowerCase:   true,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		panic(fmt.Sprintf("数据库连接失败：%s", err.Error()))
	}
	return db
}
