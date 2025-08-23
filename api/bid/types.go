package bid

import (
	"120bid/utils/api_helper"
	"120bid/utils/proxy_handler"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// SprintParams 拼接参数
func SprintParams(query, dType, startDate, endDate, lastsort string) string {
	var params string
	// 拼接查询关键字
	params = fmt.Sprintf("q=%s", query)
	// 拼接数据类型参数
	if dType != "" {
		params = fmt.Sprintf("%s%s", params, dType)
	}
	// 拼接日期范围
	if startDate != "" && endDate != "" {
		params = fmt.Sprintf("%s&date[]=%s&date[]=%s", params, startDate, endDate)
	}
	// 拼接下一页参数
	if lastsort != "" {
		params = fmt.Sprintf("%s%s", params, lastsort)
	}
	return params
}

// ExtractTextFromTagA 从a标签中提取文本内容
func ExtractTextFromTagA(content string) string {
	re := regexp.MustCompile(`<a [^>]*>([^<]+)</a>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// ParseDate 尝试解析日期字符串，如果失败则返回当前日期
func ParseDate(dateStr string) string {
	// 尝试按标准格式解析日期
	_, err := time.Parse("2006-01-02", dateStr)
	if err == nil {
		// 如果解析成功，返回原始字符串
		return dateStr
	}
	// 如果解析失败，返回当前日期
	return time.Now().Format("2006-01-02")
}

func ProcessItem(item *api_helper.Data, clientPool *proxy_handler.ProxyClientPool, wg *sync.WaitGroup) {
	defer wg.Done()

	// 检查是否存在此公告
	current := item.Find()
	if current != nil {
		// 存在公告，直接服用原数据
		item.Url = current.Url
		item.Html = current.Html
		item.Title = current.Title
		item.Status = current.Status
		item.Area = current.Area
		item.City = current.City
		item.User = current.User
		item.Date = current.Date
		return
	}

	// 从客户端池2中随机获取一个代理客户端
	client, err := clientPool.GetProxyClient()
	if err != nil {
		log.Println(err)
		return
	}

	// 获取html内容和公告原链接
	content, originalUrl, err := client.GetHtmlContentAndUrl(item.Url)
	if err != nil {
		log.Println(err)
		return
	}

	// 修改切片中的元素
	item.Url = originalUrl
	item.Html = content

	item.Create()
}
