package http

import (
	"120bid/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/supercat0867/proxyhttp"
	"golang.org/x/net/publicsuffix"
)

type ProxyFetcher struct {
	jar *cookiejar.Jar
}

func NewProxyFetcher() ProxyFetcher {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return ProxyFetcher{
		jar: jar,
	}
}

func (f *ProxyFetcher) Fetch(count int) ([]*proxyhttp.ProxyClient, error) {
	cfg := config.GetConfig()
	proxy := fmt.Sprintf("https://share.proxy.qg.net/get?key=%s&pwd=%s&num=%d&distinct=true&area=440000",
		cfg.ProxyIP.AuthKey, cfg.ProxyIP.Password, count)

	// 请求代理IP
	resp, err := http.Get(proxy)
	if err != nil {
		return nil, fmt.Errorf("代理IP请求失败：%v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("代理IP请求响应读取失败：%v", err)
	}

	// 解析响应
	var response ProxyIPResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("代理IP请求响应解析失败：%v", err)
	}

	// 检查是否成功响应
	if response.Code != "SUCCESS" {
		return nil, fmt.Errorf("代理IP请求响应失败，错误信息：%s", response.Code)
	}

	localLocation, _ := time.LoadLocation("Local")
	var clients []*proxyhttp.ProxyClient
	// 循环IP列表
	for _, data := range response.Data {
		rawURL := fmt.Sprintf("http://%s:%s@%s", cfg.ProxyIP.AuthKey, cfg.ProxyIP.Password, data.Server)
		proxyURL, _ := url.Parse(rawURL)
		deadline, _ := time.ParseInLocation(time.DateTime, data.Deadline, localLocation)

		newClient := &proxyhttp.ProxyClient{
			Client: &http.Client{
				Jar: f.jar,
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 10 * time.Second,
			},
			Expiration: deadline,
		}

		clients = append(clients, newClient)
	}
	return clients, nil
}
