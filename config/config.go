package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	// 1. 先尝试从程序所在目录加载
	exePath, err := os.Executable()
	if err != nil {
		panic(fmt.Sprintf("获取程序路径失败：%s", err.Error()))
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 2. 如果程序目录下没有，就尝试当前工作目录（IDE 调试时用）
		configPath = "config.yaml"
	}

	viper.SetConfigFile(configPath)

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
