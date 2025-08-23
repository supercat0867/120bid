package main

import (
	"120bid/api"
	"120bid/types"
	"120bid/utils/graceful"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 注册路由
	api.RegisterHandler(r)

	// goroutine 启动API服务
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", types.Conf.Server.Port),
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("监听到服务异常或主动关闭:%v", err)
		}
	}()

	// 欢迎
	graceful.Welcome()
	// 阻塞
	graceful.Shutdown(srv, time.Second*2)
}
