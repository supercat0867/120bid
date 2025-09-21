package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 配置文件结构
type Config struct {
	MySQL struct {
		User     string
		Password string
		Host     string
		Port     int
		DB       string
	}

	Params struct {
		Keywords  []string
		Status    []string
		StartDate string
		EndDate   string
	}

	ProxyIP struct {
		AuthKey  string
		Password string
	}
}

var conf *Config

// ReadConfig 读取配置文件
func ReadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("配置文件读取失败：%s", err.Error()))
	}
	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		panic(fmt.Sprintf("配置文件解析失败：%s", err.Error()))
	}
	return &conf
}

// GetConfig 获取配置文件
func GetConfig() *Config {
	if conf == nil {
		conf = ReadConfig()
	}
	return conf
}
