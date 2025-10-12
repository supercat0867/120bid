package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/supercat0867/120bid/config"
	"github.com/supercat0867/120bid/db"
	"github.com/supercat0867/120bid/http"
)

// extractTextFromTagA 从a标签中提取文本内容
func extractTextFromTagA(content string) string {
	re := regexp.MustCompile(`<a [^>]*>([^<]+)</a>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// parseDate 尝试解析日期字符串，如果失败则返回当前日期
func parseDate(dateStr string) string {
	// 尝试按标准格式解析日期
	_, err := time.Parse("2006-01-02", dateStr)
	if err == nil {
		// 如果解析成功，返回原始字符串
		return dateStr
	}
	// 如果解析失败，返回当前日期
	return time.Now().Format("2006-01-02")
}

// 初始化日志（控制台 + 文件）
func initLogger() {
	// 日志文件路径
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("创建日志目录失败: %v", err)
	}
	logFile := filepath.Join(logDir, "app.log")

	// 打开日志文件（追加写入）
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("打开日志文件失败: %v", err)
	}

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)

	// 设置日志格式（带时间、文件名、行号）
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cfg := config.GetConfig()

	db.Init()
	db.Migrate()

	initLogger()

	itemChan := make(chan http.Data, 1000)
	client := http.NewHttp()

	// 启动 worker 消费数据
	var wg sync.WaitGroup
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int, client http.Http) {
			defer wg.Done()
			for item := range itemChan {
				log.Printf("[worker-%d] 开始处理 %s", id, item.ItemRaw)

				// 获取详细内容和原链接
				content, link, err := client.GetHtmlContentAndUrl(item.Url)
				if err != nil {
					log.Printf("[worker-%d] 获取详细内容失败：%v", id, err)
					continue
				}

				area := extractTextFromTagA(item.AreaName)
				city := extractTextFromTagA(item.CityStr)

				data := db.Data{
					Url:         item.Url,
					OriginalUrl: link,
					Title:       item.ItemRaw,
					Status:      item.Status,
					Area:        area,
					Keyword:     item.Keyword,
					City:        city,
					User:        item.User,
					Html:        content,
					Date:        parseDate(item.DateStr),
				}

				if err := db.Insert(&data); err != nil {
					log.Printf("[worker-%d] 数据插入失败：%v", id, err)
				}
			}
		}(i, client)
	}

	for _, keyword := range cfg.Params.Keywords {
		client := http.NewHttp()

		data, err := client.Search(http.QueryParams{
			Query:     keyword,
			Status:    cfg.Params.Status,
			StartDate: cfg.Params.StartDate,
			EndDate:   cfg.Params.EndDate,
		})
		if err != nil {
			log.Printf("关键词【%s】查询失败：%v", keyword, err)
			continue
		}

		for _, v := range data {
			itemChan <- v
		}
	}

	close(itemChan)

	wg.Wait()

	log.Println("所有任务处理完成")
}
