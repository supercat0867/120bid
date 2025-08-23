package types

import (
	"120bid/config"

	"gorm.io/gorm"
)

var DB *gorm.DB
var Conf *config.Config

// SearchAllResponse 120bid全文搜索开启响应
type SearchAllResponse struct {
	Code int `json:"code"`
}

// ProxyIPResponse 代理IP请求响应
type ProxyIPResponse struct {
	Code      string        `json:"code"`
	Data      []ProxyIPData `json:"data"`
	RequestID string        `json:"request_id"`
}

type ProxyIPData struct {
	ProxyIP  string `json:"proxy_ip"`
	Server   string `json:"server"`
	Area     string `json:"area"`
	ISP      string `json:"isp"`
	Deadline string `json:"deadline"`
}
