package http

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/supercat0867/120bid/constdef"

	"github.com/PuerkitoBio/goquery"
	"github.com/supercat0867/proxyhttp"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
)

type Http interface {
	// Search 搜索
	Search(params QueryParams) ([]Data, error)
	// GetHtmlContentAndUrl 获取html内容，公告原链接
	GetHtmlContentAndUrl(target string) (string, string, error)
}

type httpImpl struct {
	client *proxyhttp.Client
}

func NewHttp() Http {
	fetcher := NewProxyFetcher()
	pool := proxyhttp.NewHttpPool(10, &fetcher)
	client := proxyhttp.NewClient(pool)
	return &httpImpl{
		client: client,
	}
}

// Search 搜索
func (h *httpImpl) Search(params QueryParams) ([]Data, error) {
	var name, value string
	var pit string
	var lastsort []interface{}
	var err error

	// 首次必须打码验证
	if err = h.crackCaptcha(20); err != nil {
		return nil, err
	}

	// 开启全文检索
	if err = h.openSearchAll(); err != nil {
		return nil, err
	}

	refererParams := url.Values{}
	refererParams.Set("q", params.Query)
	referer := fmt.Sprintf("https://www.120bid.com/search?%s", refererParams.Encode())

	bids := make([]Data, 0)

	// 获取name和value
	name, value, err = h.getNameAndValue(params.Query)
	if err != nil {
		return nil, fmt.Errorf("获取name和value失败：%v", err)
	}

	// 定义一个发送请求的内部函数
	doSearchPage := func(page int) (*SearchRsp, error) {
		log.Printf("正在获取关键词【%s】第%d页数据...", params.Query, page)

		form := url.Values{}
		form.Add("q", params.Query)
		if params.StartDate != "" && params.EndDate != "" {
			form.Add("date[]", params.StartDate)
			form.Add("date[]", params.EndDate)
		}
		for _, v := range params.Status {
			form.Add("status[]", v)
		}
		form.Add(name, value)
		if pit != "" {
			form.Add("pit", pit)
		}
		for _, v := range lastsort {
			switch vv := v.(type) {
			case json.Number:
				form.Add("lastsort[]", vv.String())
			case string:
				form.Add("lastsort[]", vv)
			}
		}

		link := fmt.Sprintf("https://www.120bid.com/ajax/search?t=%d", time.Now().UnixMilli())
		req, err := http.NewRequest("POST", link, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("构造请求失败: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://www.120bid.com")
		req.Header.Set("Referer", referer)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		resp, err := h.client.DoWithRetry(req, 3, 2*time.Second)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}

		var response SearchRsp
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
		}
		return &response, nil
	}

	for i := 1; i <= 20; i++ {
		const maxRetry = 3
		var response *SearchRsp
		var err error

		for retry := 1; retry <= maxRetry; retry++ {
			response, err = doSearchPage(i)
			if err != nil {
				log.Printf("请求第%d页失败（第%d次尝试）: %v", i, retry, err)
				time.Sleep(time.Second)
				continue
			}

			// 如果返回 401，触发验证码破解并重试
			if response.Status == 401 {
				log.Printf("检测到验证码过期（第%d页，第%d次尝试），开始重新识别...", i, retry)
				if err := h.crackCaptcha(20); err != nil {
					log.Printf("验证码识别失败: %v", err)
					time.Sleep(2 * time.Second)
					continue
				}
				time.Sleep(1 * time.Second)
				continue
			}

			// 正常响应则跳出重试循环
			break
		}

		// 为下一页设置查询参数
		for k, v := range response.Extras {
			name = k
			value = v
		}
		pit = response.Pit
		lastsort = response.LastSort

		for index, v := range response.Data {
			response.Data[index].Url = "https://www.120bid.com" + v.Url
			response.Data[index].Keyword = params.Query
		}

		bids = append(bids, response.Data...)

		if response.Pit == "" {
			log.Println("暂无更多数据")
			break
		}

		// 休眠
		time.Sleep(2 * time.Second)
	}

	return bids, nil
}

// getNameAndValue 获取名称和值
func (h *httpImpl) getNameAndValue(keyword string) (string, string, error) {
	params := url.Values{}
	params.Set("q", keyword)
	target := fmt.Sprintf("https://www.120bid.com/search?%s", params.Encode())

	// 构造请求
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return "", "", err
	}

	// 设置请求头部信息
	req.Header.Set("Referer", "https://www.120bid.com/")
	req.Header.Set("User-Agent", constdef.UserAgent)

	// 发送请求
	resp, err := h.client.DoWithRetry(req, 3, 2*time.Second)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("页面解析失败：%v", err)
	}

	name, value, found := getNameAndValueFromNode(doc, "hidden", "token")
	if found {
		return name, value, nil
	} else {
		return "", "", fmt.Errorf("Name,Value 定位失败")
	}
}

// 递归查找并提取指定类型的元素属性
func getNameAndValueFromNode(n *html.Node, inputType, inputID string) (name, value string, found bool) {
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
		name, value, found = getNameAndValueFromNode(c, inputType, inputID)
		if found {
			return
		}
	}
	return
}

// captchaVerifyResp 验证码验证接口响应
type captchaVerifyResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// crackCaptcha 破解验证码并提交
func (h *httpImpl) crackCaptcha(maxRetry int) error {
	if maxRetry <= 0 {
		maxRetry = 5 // 默认最多尝试 5 次
	}
	var lastErr error

	for attempt := 1; attempt <= maxRetry; attempt++ {
		log.Printf("第 %d 次尝试识别验证码...\n", attempt)

		// 调用 OCR 识别
		codeText, err := h.verifyCaptcha()
		if err != nil {
			lastErr = fmt.Errorf("识别验证码出错: %w", err)
			continue
		}
		log.Println("识别结果:", codeText)

		// POST 提交验证码
		api := "https://www.120bid.com/captcha/verify"
		data := url.Values{}
		data.Set("code", strings.TrimSpace(codeText))

		req, err := http.NewRequest("POST", api, bytes.NewBufferString(data.Encode()))
		if err != nil {
			lastErr = fmt.Errorf("构造请求失败: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("User-Agent", constdef.UserAgent)
		req.Header.Set("Referer", "https://www.120bid.com/search")
		req.Header.Set("Origin", "https://www.120bid.com")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		resp, err := h.client.DoWithRetry(req, 3, 3*time.Second)
		if err != nil {
			lastErr = fmt.Errorf("提交验证码失败: %w", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result captchaVerifyResp
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %w, body: %s", err, string(body))
			continue
		}

		if result.Code == 1 {
			log.Println("验证成功:", result.Msg)
			return nil
		}

		log.Println("验证失败:", result.Msg)
		// 短暂等待再重试（避免请求过快）
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("验证码多次识别失败: %v", lastErr)
}

type ocrResult struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

// verifyCaptcha 识别验证码
func (h *httpImpl) verifyCaptcha() (string, error) {
	captchaPath, err := h.getCaptcha()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(captchaPath)
	}()

	// 获取 exe 所在目录
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取 exe 路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// 默认 Python 脚本路径
	pythonScript := filepath.Join(exeDir, "captcha_ocr.py")

	// 如果 exeDir 下不存在，再尝试当前工作目录（IDE 运行时）
	if _, err := os.Stat(pythonScript); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		pythonScript = filepath.Join(cwd, "captcha_ocr.py")
	}

	// 选择 Python 可执行文件
	pythonBin := "python"
	if _, err := exec.LookPath("python3"); err == nil {
		pythonBin = "python3"
	}

	// 设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, pythonBin, pythonScript, captchaPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("调用 OCR 超时")
	}
	if err != nil {
		// 返回 stderr 便于排查
		return "", fmt.Errorf("调用 OCR 脚本失败: %w, stderr: %s", err, stderr.String())
	}

	// 解析 OCR 的 JSON 输出
	var res ocrResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		// 有可能脚本直接输出裸文本，作为兜底：尝试按行取第一个非空作为结果
		trimOut := bytes.TrimSpace(stdout.Bytes())
		if len(trimOut) > 0 {
			// 直接认为是识别字符串
			return string(trimOut), nil
		}
		return "", fmt.Errorf("解析 OCR 输出失败: %w", err)
	}

	if res.Error != "" {
		return "", fmt.Errorf("OCR 脚本返回错误: %s", res.Error)
	}
	return res.Text, nil
}

// getCaptcha 获取验证码
func (h *httpImpl) getCaptcha() (string, error) {
	// 获取当前时间戳
	getUnixNow := strconv.Itoa(int(time.Now().UnixNano() / 1000000))
	// 拼接API
	api := fmt.Sprintf("https://www.120bid.com/captcha/image?t=%s", getUnixNow)

	// 构造请求
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return "", err
	}

	// 设置请求头部信息
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", constdef.UserAgent)
	req.Header.Set("Referer", "https://www.120bid.com/search")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// 发送请求
	resp, err := h.client.DoWithRetry(req, 3, 3*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "captcha-*.jpeg")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(body); err != nil {
		// 尝试删除临时文件再返回
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	return tmpFile.Name(), nil
}

// openSearchAll 开启全文搜索
func (h *httpImpl) openSearchAll() error {
	// 获取当前时间戳
	getUnixNow := strconv.Itoa(int(time.Now().UnixNano() / 1000000))
	// 拼接API
	api := fmt.Sprintf("https://www.120bid.com/ajax/match?way=1&t=%s", getUnixNow)

	// 构造请求
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return err
	}

	// 设置请求头部信息
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("User-Agent", constdef.UserAgent)
	req.Header.Set("Referer", "https://www.120bid.com/search")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// 发送请求
	resp, err := h.client.DoWithRetry(req, 3, 3*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 解析响应
	var response SearchAllResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("全文搜索响应解析失败：%v, resp:%s", err, string(body))
	}

	if response.Code != 1 {
		return fmt.Errorf("错误码: %v", response.Code)
	}

	return nil
}

// GetHtmlContentAndUrl 获取html内容，公告原链接
func (h *httpImpl) GetHtmlContentAndUrl(target string) (string, string, error) {
	// 构造请求
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return "", "", err
	}

	// 设置请求头部信息
	req.Header.Set("Host", "www.120bid.com")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("User-Agent", constdef.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Referer", "https://www.120bid.com/search")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// 发送请求
	resp, err := h.client.DoWithRetry(req, 3, 5*time.Second)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var body []byte

	// 检查 Content-Encoding
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", "", err
		}
		defer gzipReader.Close()

		body, err = io.ReadAll(gzipReader)
		if err != nil {
			return "", "", err
		}
	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", "", err
		}
	}

	var content, originalLink string

	content = string(body)
	// 检查字符数是否过多
	if len(content) <= 80000 {
		divHTML, err := h.extractElementByID(content, "article", "content")
		if err != nil {
			log.Printf("公告链接【%s】匹配失败，已将html字段设置为空", target)
			content = ""
		} else {
			content = divHTML

			// 清洗
			content, _ = CleanAnnouncementHTML(content)

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
			content = phoneRegex.ReplaceAllStringFunc(content, replaceFunc)
		}
	} else {
		log.Printf("公告链接【%s】字符数过多，已将html字段设置为空，字符数：%d", target, len(content))
		content = ""
	}

	// 获取原链接
	originalLinkRegex := regexp.MustCompile(`<div class="view-url">\s*<a [^>]*?data-view="([^"]+)"[^>]*?>`)
	matches := originalLinkRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		log.Printf("公告链接【%s】未匹配到原链接", target)
	} else {
		base64Str := matches[1]
		// Base64 解码
		decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			log.Printf("公告链接【%s】原链接base64解码失败", target)
		} else {
			originalLink = string(decodedBytes)
		}
	}

	return content, originalLink, nil
}

// extractElementByID 从内容中提取指定标签+id 的完整 HTML（包含自身标签）
func (h *httpImpl) extractElementByID(content, tagName, id string) (string, error) {
	// 解析 HTML
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("HTML 解析失败: %v", err)
	}

	// 深度优先搜索目标节点
	var target *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tagName {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					target = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
			if target != nil {
				return
			}
		}
	}
	f(doc)

	if target == nil {
		return "", fmt.Errorf("未找到 <%s id=\"%s\"> 节点", tagName, id)
	}

	// 序列化节点为 HTML
	var buf bytes.Buffer
	if err := html.Render(&buf, target); err != nil {
		return "", fmt.Errorf("节点序列化失败: %v", err)
	}

	return buf.String(), nil
}

// CleanAnnouncementHTML 清理公告 HTML：
// 1. 删除包含 "本公告地址" 的 <p> 元素
// 2. 删除最后一个 div.view-url（常见的原文链接块）
// 3. 将 href 中包含 "www.120bid.com" 的 <a> 标签去掉，但保留其内部文本
// 4. 对 span.span-link 元素的 data-url 做 base64 解码并直接覆盖 data-url 的值（不改变标签类型）
func CleanAnnouncementHTML(input string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	// 1. 删除包含 "本公告地址" 的 <p>
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "本公告地址") {
			s.Remove()
		}
	})

	// 2. 删除最后一个 div.view-url（可能存在多个，用最后那个）
	doc.Find("div.view-url").Last().Remove()

	// 3. 对所有 a 标签，若 href 包含 www.120bid.com，则替换为纯文本（保留内部文本）
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok && strings.Contains(href, "www.120bid.com") {
			// 获取文本并进行 HTML 转义，避免插入未转义的 html
			text := html.EscapeString(s.Text())
			// 用转义文本替换原节点（ReplaceWithHtml 会把字符串作为 HTML 插入）
			// 因为是转义过的所以等同于插入纯文本
			_ = s.ReplaceWithHtml(text)
		}
	})

	// 4. 解码 span.span-link 的 data-url 并覆盖 data-url 值
	doc.Find("span.span-link").Each(func(i int, s *goquery.Selection) {
		if encoded, ok := s.Attr("data-url"); ok && encoded != "" {
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				// 解码失败则跳过但不影响其它处理
				return
			}
			s.SetAttr("data-url", string(decoded))
		}
	})

	// 最后返回第一个 <article class="content"> 的完整 outer HTML（包含 article 标签）
	// 如果没有 article 节点，则返回整个文档的 HTML（fallback）
	articleSel := doc.Find("article.content").First()
	if articleSel.Length() == 0 {
		// fallback: 返回整个文档 html
		htmlStr, err := doc.Html()
		if err != nil {
			return "", err
		}
		return htmlStr, nil
	}

	// 把 article 节点渲染为 outer HTML
	var buf bytes.Buffer
	for _, n := range articleSel.Nodes {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
