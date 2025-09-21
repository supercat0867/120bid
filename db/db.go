package db

import (
	"120bid/config"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Init 初始化数据库
func Init() {
	cfg := config.GetConfig()

	once.Do(func() {
		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				Colorful:                  true,
			},
		)

		var err error
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.DB)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger:      newLogger,
			PrepareStmt: true,
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		})
		if err != nil {
			log.Fatalf("数据库连接失败: %v", err)
		}
	})
}

// GetDB 获取 GORM 实例
func GetDB() *gorm.DB {
	if db == nil {
		log.Fatal("数据库未初始化，请先调用 db.Init()")
	}
	return db
}

// Data 数据表结构
type Data struct {
	ID          uint   `gorm:"primaryKey"`
	Url         string `gorm:"unique"`
	OriginalUrl string // 原链接
	Title       string
	Status      string
	Area        string
	City        string
	User        string
	Date        string
	Html        string `gorm:"type:longtext"`
	Keyword     string // 关键词
}

// Migrate 迁移数据库表结构
func Migrate() {
	db := GetDB()
	if err := db.AutoMigrate(&Data{}); err != nil {
		log.Fatalf("数据库表迁移失败: %v", err)
	}
}

// Insert 批量插入数据
func Insert(bids *Data) error {
	if err := db.Create(&bids).Error; err != nil {
		return err
	}
	return nil
}
