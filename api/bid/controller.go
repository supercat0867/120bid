package bid

import (
	"120bid/utils/api_helper"
	"120bid/utils/proxy_handler"
	"120bid/utils/validator"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var ClientPool1 *proxy_handler.ProxyClientPool
var ClientPool2 *proxy_handler.ProxyClientPool

func Fetch120bidAPI(c *gin.Context) {
	// 查询关键字
	q := c.Query("q")
	if q == "" {
		api_helper.BadRequestHandler(c, errors.New("查询关键字q不能为空！"))
		return
	}

	// 数据类型
	t := c.QueryArray("t")
	var dtype string
	for _, item := range t {
		switch item {
		case "1":
			dtype += "&status[]=招标预告"
		case "2":
			dtype += "&status[]=招标公告"
		case "3":
			dtype += "&status[]=招标变更"
		case "4":
			dtype += "&status[]=招标结果"
		case "5":
			dtype += "&status[]=合同验收"
		case "6":
			dtype += "&status[]=招标信用"
		case "7":
			dtype += "&status[]=招标预告-意向"
		case "8":
			dtype = "&status[]=招标预告-需求"
		case "9":
			dtype += "&status[]=招标预告-意见"
		case "10":
			dtype += "&status[]=招标预告-预告"
		case "11":
			dtype += "&way[]=公开招标"
		case "12":
			dtype += "&way[]=邀请招标"
		case "13":
			dtype += "&way[]=竞争性谈判"
		case "14":
			dtype += "&way[]=竞争性磋商"
		case "15":
			dtype += "&way[]=单一来源采购"
		case "16":
			dtype += "&way[]=询价"
		case "17":
			dtype += "&way[]=网上竞价"
		case "18":
			dtype += "&way[]=其他"
		case "19":
			dtype += "&status[]=招标公告-重招"
		case "20":
			dtype += "&status[]=招标公告-第N次"
		case "21":
			dtype += "&status[]=招标变更-变更"
		case "22":
			dtype += "&status[]=招标变更-补充"
		case "23":
			dtype += "&status[]=招标变更-澄清"
		case "24":
			dtype += "&status[]=招标变更-延期"
		case "25":
			dtype += "&status[]=招标变更-终止"
		case "26":
			dtype += "&status[]=招标结果-结果"
		case "27":
			dtype += "&status[]=招标结果-中标"
		case "28":
			dtype += "&status[]=招标结果-成交"
		case "29":
			dtype += "&status[]=招标结果-废标"
		case "30":
			dtype += "&status[]=招标结果-流标"
		case "31":
			dtype += "&status[]=合同验收-合同"
		case "32":
			dtype += "&status[]=合同验收-验收"
		case "33":
			dtype += "&status[]=招标信用-违规"
		case "34":
			dtype += "&status[]=招标信用-违约"
		case "35":
			dtype += "&status[]=招标信用-处罚"
		default:
			dtype += ""
		}
	}

	log.Printf("开始查询：查询关键词：%s 公告类型：%s\n", q, dtype)

	// 开始日期、结束日期
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	// 检查日期参数是否合法
	if !validator.IsDateValid(startDate) && !validator.IsDateValid(endDate) {
		api_helper.BadRequestHandler(c, errors.New("日期参数格式错误，格式必须为2024-01-19！"))
		return
	}
	// 下一页参数
	lastsort := c.Query("ls")

	// 从客户端池1中随机获取一个代理客户端
	client, err := ClientPool1.GetProxyClient()
	if err != nil {
		api_helper.InternalServerHandler(c, err)
		return
	}

	// 检查客户端是否已开启全文搜索
	if !client.Status {
		// 未开启全文搜索，则提交开启
		err = client.OpenSearchAll()
		if err != nil {
			api_helper.InternalServerHandler(c, err)
			return
		}
	}

	// 获取name和value
	name, value, err := client.GetNameAndValue()
	if err != nil {
		api_helper.InternalServerHandler(c, err)
		return
	}

	// 获取当前毫秒级时间戳
	getUnixNow := strconv.Itoa(int(time.Now().UnixNano() / 1000000))
	// 拼接参数
	params := fmt.Sprintf("%s&t=%s&%s=%s", SprintParams(q, dtype, startDate, endDate, lastsort), getUnixNow, name, value)

	// 调用120bidAPI
	response, err := client.Fetch120bidAPI(params)
	if err != nil {
		log.Println(err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 循环遍历构造数据
	var items []*api_helper.Data

	for _, data := range response.Data {
		wg.Add(1)
		go func(data proxy_handler.Data) {
			defer wg.Done()

			// 公告链接
			target := fmt.Sprintf("https://www.120bid.com%s", data.Url)

			// 从<a></a>标签中提取出地区、城市文本内容
			area := ExtractTextFromTagA(data.AreaName)
			city := ExtractTextFromTagA(data.CityStr)

			// 构造基础item
			item := api_helper.Data{
				Url:    target,
				Title:  data.ItemRaw,
				Status: data.Status,
				Area:   area,
				City:   city,
				User:   data.User,
				Date:   ParseDate(data.DateStr),
			}

			mu.Lock()
			items = append(items, &item)
			mu.Unlock()
		}(data)
	}
	wg.Wait()

	// 循环遍历items,访问目标网站获取html和其他信息
	for i := range items {
		wg.Add(1)
		go ProcessItem(items[i], ClientPool2, &wg)
	}
	// 等待第二组 goroutine 完成
	wg.Wait()

	// 检查是否存在下一页数据
	var nextLastSort string
	for _, value := range response.Lastsort {
		switch v := value.(type) {
		case float64:
			// 对浮点数值进行格式化
			nextLastSort += "&lastsort[]=" + fmt.Sprintf("%.0f", v)
		case string:
			// 对字符串直接使用原值
			nextLastSort += "&lastsort[]=" + v
		}
	}

	// 响应结构
	var currentResponse = api_helper.Response{
		Status:   200,
		Message:  "success",
		ErrCode:  0,
		Count:    len(items),
		LastSort: nextLastSort,
		Items:    items,
	}

	log.Println("查询结束")
	c.JSON(http.StatusOK, currentResponse)
}
