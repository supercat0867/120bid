package api

import (
	"120bid/api/bid"
	"120bid/config"
	"120bid/types"
	"120bid/utils/database_handler"
	"120bid/utils/proxy_handler"

	"github.com/gin-gonic/gin"
)

// InitialConfAndDB 初始化数据库和配置文件
func InitialConfAndDB() {
	types.Conf = config.ReadConfig()
	types.DB = database_handler.NewMySQL(types.Conf)
	database_handler.Migrate(types.DB)
	bid.ClientPool1 = proxy_handler.NewProxyClientPool(types.Conf.ProxyPool.ProxyPool1)
	bid.ClientPool2 = proxy_handler.NewProxyClientPool(types.Conf.ProxyPool.ProxyPool2)
}

// RegisterHandler 注册路由
func RegisterHandler(r *gin.Engine) {
	InitialConfAndDB()
	group := r.Group("/api")
	group.GET("/get", bid.Fetch120bidAPI)
}
