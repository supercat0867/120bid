package proxy_handler

import (
	"120bid/types"
	"120bid/utils/http_handler"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

// ProxyClient 表示带有过期时间的 HTTP 客户端
type ProxyClient struct {
	Client     *http.Client
	IP         string
	Expiration time.Time
	Status     bool
}

// ProxyClientPool 管理代理客户端池的结构体
type ProxyClientPool struct {
	clients  []*ProxyClient
	maxCount int
	mu       sync.Mutex
}

// A120binResponse 120bid的API响应结构体
type A120binResponse struct {
	Status   int           `json:"status"`
	Msg      string        `json:"msg"`
	Data     []Data        `json:"data"`
	Pit      string        `json:"pit"`
	Lastsort []interface{} `json:"lastsort"`
}
type Data struct {
	Url      string `json:"url"`
	ItemRaw  string `json:"itemRaw"`
	Item     string `json:"item"`
	Class    string `json:"class"`
	Status   string `json:"status"`
	AreaName string `json:"areaName"`
	User     string `json:"user"`
	UserLink string `json:"userLink"`
	CityStr  string `json:"cityStr"`
	Content  string `json:"content"`
	DateStr  string `json:"dateStr"`
}

// NewProxyClientPool 创建一个新的 ProxyClientPool
func NewProxyClientPool(maxCount int) *ProxyClientPool {
	return &ProxyClientPool{
		maxCount: maxCount,
	}
}

// removeProxyClient 从池中删除指定的代理客户端
func (p *ProxyClientPool) removeProxyClient(client *ProxyClient) {
	for i, c := range p.clients {
		if c == client {
			p.clients = append(p.clients[:i], p.clients[i+1:]...)
			break
		}
	}
}

// GetProxyClient 从客户端池中获取一个代理客户端，检查是否过期，如果过期则重新获取
func (p *ProxyClientPool) GetProxyClient() (*ProxyClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		// 如果池子为空或所有客户端都过期了，先补充池子
		if len(p.clients) == 0 {
			newClients, err := p.fetchNewProxyClient(p.maxCount)
			if err != nil {
				return nil, err // 如果补充失败，则返回错误
			}
			p.clients = append(p.clients, newClients...)
		}

		//// 从当前池子中随机选择一个代理客户端
		index := rand.Intn(len(p.clients))
		proxyClient := p.clients[index]
		//log.Printf("当前索引：%d", index)
		//p.currentIndex = (p.currentIndex + 1) % len(p.clients) // 更新索引，如果到达末尾则从头开始

		// 检查代理客户端是否过期
		if proxyClient.Expiration.Before(time.Now()) {
			log.Printf("IP %s 过期，已从池子中删除\n", proxyClient.IP)
			p.removeProxyClient(proxyClient) // 删除过期的代理客户端
			//p.currentIndex = p.currentIndex - 1
			//if p.currentIndex < 0 {
			//	p.currentIndex = len(p.clients) - 1 // 如果索引变成负数，调整为最后一个客户端的索引
			//}
		} else {
			return proxyClient, nil // 返回有效的代理客户端
		}
	}
}

// 获取指定数量新的代理客户端
func (p *ProxyClientPool) fetchNewProxyClient(count int) ([]*ProxyClient, error) {
	authKey := types.Conf.ProxyIP.AuthKey
	password := types.Conf.ProxyIP.Password
	proxyAPI := fmt.Sprintf("https://share.proxy.qg.net/get?key=%s&pwd=%s&num=%d&distinct=true&area=440000", authKey, password, count)

	// 请求代理IP
	resp, err := http.Get(proxyAPI)
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
	var response types.ProxyIPResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("代理IP请求响应解析失败：%v", err)
	}

	// 检查是否成功响应
	if response.Code != "SUCCESS" {
		return nil, fmt.Errorf("代理IP请求响应失败，错误信息：%s", response.Code)
	}

	localLocation, _ := time.LoadLocation("Local")
	var clients []*ProxyClient
	// 循环IP列表
	for _, data := range response.Data {
		rawURL := fmt.Sprintf("http://%s:%s@%s", authKey, password, data.Server)
		proxyURL, _ := url.Parse(rawURL)
		deadline, _ := time.ParseInLocation("2006-01-02 15:04:05", data.Deadline, localLocation)

		log.Printf("获取新的代理IP：%s，过期时间：%s\n", data.ProxyIP, data.Deadline)

		// 创建一个 cookie jar 来存储 cookies
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, err
		}
		// 创建一个 HTTP 代理客户端
		newClient := &ProxyClient{
			Client: &http.Client{
				Jar: jar,
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
			},
			IP:         data.ProxyIP,
			Expiration: deadline,
		}
		clients = append(clients, newClient)
	}
	return clients, nil
}

// OpenSearchAll 开启全文搜索
func (pc *ProxyClient) OpenSearchAll() error {
	// 获取当前时间戳
	getUnixNow := strconv.Itoa(int(time.Now().UnixNano() / 1000000))
	// 拼接API
	api := fmt.Sprintf("https://www.120bid.com/ajax/match?way=1&t=%s", getUnixNow)

	// 构造请求
	req, err := http_handler.NewRequest("GET", api, nil)
	if err != nil {
		return err
	}

	// 设置请求头部信息
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1")
	req.Header.Set("Referer", "https://www.120bid.com/search")

	// 发送请求
	resp, err := pc.Client.Do(req)
	if err != nil {
		return fmt.Errorf("全文搜索开启请求错误：%v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("全文搜索响应读取失败：%v", err)
	}
	// 解析响应
	fmt.Println(string(body))
	var response types.SearchAllResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("全文搜索响应解析失败：%v", err)
	}
	if response.Code != 1 {
		return fmt.Errorf("全文搜索开启失败，错误码%v", response.Code)
	}
	pc.Status = true
	return nil
}

// GetNameAndValue 获取name和value
func (pc *ProxyClient) GetNameAndValue() (string, string, error) {
	// 构造请求
	req, err := http_handler.NewRequest("GET", "https://www.120bid.com/search", nil)
	if err != nil {
		return "", "", err
	}

	// 设置请求头部信息
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Host", "www.120bid.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.120bid.com")
	req.Header.Set("Sec-Fetch-Dest", "document")

	// 发送请求
	resp, err := pc.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Name、Value请求失败：%v", err)
	}
	defer resp.Body.Close()

	// 解析 HTML响应
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("搜索页面解析失败：%v", err)
	}

	name, value, found := findAndExtractInputAttributes(doc, "hidden", "token")
	if found {
		return name, value, nil
	} else {
		return "", "", fmt.Errorf("Name、Value元素定位失败！")
	}
}

// Fetch120bidAPI 调用120bidAPI
func (pc *ProxyClient) Fetch120bidAPI(params string) (*A120binResponse, error) {
	// 构造请求
	req, err := http_handler.NewRequest("POST", "https://www.120bid.com/ajax/search", strings.NewReader(params))
	if err != nil {
		return nil, err
	}

	// 设置请求头部信息
	req.Header.Set("Host", "www.120bid.com")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1")
	req.Header.Set("Referer", "https://www.120bid.com/search")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// 发送请求
	resp, err := pc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("120bidAPI请求失败：%v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("120bidAPI响应读取失败：%v", err)
	}

	// 解析响应
	var response A120binResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("120bidAPI响应解析失败：%v", err)
	}
	return &response, nil
}

// GetHtmlContentAndUrl 获取html内容，公告原链接
func (pc *ProxyClient) GetHtmlContentAndUrl(targetUrl string) (string, string, error) {
	// 构造请求
	req, err := http_handler.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return "", "", err
	}

	// 设置请求头部信息
	req.Header.Set("Host", "www.120bid.com")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Referer", "https://www.120bid.com/search")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")

	// 发送请求
	resp, err := pc.Client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("公告内容请求失败：%v", err)
	}
	defer resp.Body.Close()

	var body []byte

	// 检查 Content-Encoding
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", "", fmt.Errorf("解压 Gzip 失败：%v\n", err)
		}
		defer gzipReader.Close()

		body, err = io.ReadAll(gzipReader)
		if err != nil {
			return "", "", fmt.Errorf("读取解压后的数据失败：%v\n", err)
		}
	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", "", fmt.Errorf("公告内容响应读取失败：%v\n", err)
		}
	}

	var htmlContent, originalLink string

	htmlContent = string(body)
	// 检查字符数是否过多
	if len(htmlContent) <= types.Conf.Misc.HtmlMaxSize {
		// 解析html
		doc, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
			return "", "", fmt.Errorf("公告内容解析失败：%v", err)
		}
		divHTML, found := findFirstDiv(doc)
		if !found {
			log.Println("公告HTML匹配失败,已将HTML字段设置为空")
			htmlContent = ""
		} else {
			htmlContent = divHTML
			// 替换函数，将base编码的手机号解码
			phoneRegex := regexp.MustCompile(`<span class="phone" data-txt="(.+?)">.*?https://www.120bid.com</span>`)
			replaceFunc := func(s string) string {
				match := phoneRegex.FindStringSubmatch(s)
				if len(match) < 2 {
					return s
				}
				// Base64解码
				decoded, err := base64.StdEncoding.DecodeString(match[1])
				if err != nil {
					return s
				}
				// 创建新的span标签
				return fmt.Sprintf(`<span class="phone">%s</span>`, decoded)
			}
			htmlContent = phoneRegex.ReplaceAllStringFunc(htmlContent, replaceFunc)
		}
	} else {
		log.Printf("公告HTML字符数过多，已将HTML字段设置为空。字符数：%d\n", len(htmlContent))
		htmlContent = ""
	}

	// 匹配原链接
	originalLinkRegex := regexp.MustCompile(`<div class="view-url">\s*<a [^>]*?data-view="([^"]+)"[^>]*?>`)
	matches := originalLinkRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		log.Println("未匹配到原链接，本字段将用120bid链接替代")
		originalLink = targetUrl
	} else {
		base64Str := matches[1]
		// Base64 解码
		decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			log.Println("原链接解密失败，本字段将用120bid链接替代")
			originalLink = targetUrl
		}
		originalLink = string(decodedBytes)
	}

	return htmlContent, originalLink, nil
}

func findAndExtractInputAttributes(n *html.Node, inputType, inputID string) (name, value string, found bool) {
	if n.Type == html.ElementNode && n.Data == "input" {
		var isMatchedType, isMatchedID bool
		for _, a := range n.Attr {
			if a.Key == "type" && a.Val == inputType {
				isMatchedType = true
			}
			if a.Key == "id" && a.Val == inputID {
				isMatchedID = true
			}
			if a.Key == "name" {
				name = a.Val
			}
			if a.Key == "value" {
				value = a.Val
			}
		}
		if isMatchedType && isMatchedID {
			found = true
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		name, value, found = findAndExtractInputAttributes(c, inputType, inputID)
		if found {
			return
		}
	}
	return
}

func findFirstDiv(n *html.Node) (string, bool) {
	// 检查当前节点是否是<article>
	if n.Type == html.ElementNode && n.Data == "article" {
		// 在<article>内查找第一个<div>
		return findDivInArticle(n)
	}
	// 对所有子节点递归搜索
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		divHTML, found := findFirstDiv(c)
		if found {
			return divHTML, true
		}
	}
	return "", false
}

func findDivInArticle(n *html.Node) (string, bool) {
	if n.Type == html.ElementNode && n.Data == "div" {
		var buf bytes.Buffer
		html.Render(&buf, n)
		return buf.String(), true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		divHTML, found := findDivInArticle(c)
		if found {
			return divHTML, true
		}
	}
	return "", false
}
