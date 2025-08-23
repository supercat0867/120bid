package graceful

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Shutdown 停止服务
func Shutdown(instance *http.Server, timeout time.Duration) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := instance.Shutdown(ctx); err != nil {
		log.Fatal("服务关闭失败：", err)
	}
	select {
	case <-ctx.Done():
		log.Println("超时")
	}
	log.Println("服务已关闭")
}

// Welcome 欢迎
func Welcome() {
	author := "supercat0867"
	projectName := "120bid-API v2.0"
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	fmt.Printf("=======================================\n")
	fmt.Printf("  %s - %s\n", projectName, author)
	fmt.Printf("  启动时间: %s\n", currentTime)
	fmt.Printf("  欢迎使用 %s!\n", projectName)
	fmt.Printf("=======================================\n")
}
