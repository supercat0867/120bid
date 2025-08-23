package http_handler

import (
	"fmt"
	"io"
	"net/http"
)

// NewRequest 实例化HTTP请求
func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("请求构建失败：%v", err)
	}
	// 添加常规请求头
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Referer", "https://www.120bid.com/search")
	return req, nil
}
