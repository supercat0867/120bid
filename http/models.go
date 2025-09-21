package http

import (
	"bytes"
	"encoding/json"
)

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

// SearchAllResponse 120bid全文搜索开启响应
type SearchAllResponse struct {
	Code int `json:"code"`
}

type QueryParams struct {
	Query     string // 关键词
	Status    []string
	StartDate string // 开始时间
	EndDate   string // 结束时间
}

// SearchRsp 搜索结果
type SearchRsp struct {
	Status   int               `json:"status"`
	Msg      string            `json:"msg"`
	Count    int               `json:"count"`
	Data     []Data            `json:"data"`
	Size     int               `json:"size"`
	Pit      string            `json:"pit"`
	LastSort []interface{}     `json:"lastsort"`
	Extras   map[string]string `json:"-"`
}

type Data struct {
	Url       string      `json:"url"`
	ItemRaw   string      `json:"itemRaw"`
	Item      string      `json:"item"`
	Class     string      `json:"class"`
	Status    string      `json:"status"`
	AreaName  string      `json:"areaName"`
	CityStr   string      `json:"cityStr"`
	User      string      `json:"user,omitempty"`
	UserLink  string      `json:"userLink,omitempty"`
	BudgetStr string      `json:"budgetStr,omitempty"`
	Content   interface{} `json:"content"`
	DateStr   string      `json:"dateStr"`
	Winner    string      `json:"winner,omitempty"`
	PriceStr  string      `json:"priceStr,omitempty"`
	Keyword   string      `json:"-"` // 查询关键词
}

func (s *SearchRsp) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	raw := make(map[string]json.RawMessage)
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	// 固定字段解出来
	if v, ok := raw["status"]; ok {
		_ = json.Unmarshal(v, &s.Status)
		delete(raw, "status")
	}
	if v, ok := raw["msg"]; ok {
		_ = json.Unmarshal(v, &s.Msg)
		delete(raw, "msg")
	}
	if v, ok := raw["count"]; ok {
		_ = json.Unmarshal(v, &s.Count)
		delete(raw, "count")
	}
	if v, ok := raw["data"]; ok {
		_ = json.Unmarshal(v, &s.Data)
		delete(raw, "data")
	}
	if v, ok := raw["size"]; ok {
		_ = json.Unmarshal(v, &s.Size)
		delete(raw, "size")
	}
	if v, ok := raw["pit"]; ok {
		_ = json.Unmarshal(v, &s.Pit)
		delete(raw, "pit")
	}
	if v, ok := raw["lastsort"]; ok {
		var tmp []interface{}
		dec2 := json.NewDecoder(bytes.NewReader(v))
		dec2.UseNumber()
		_ = dec2.Decode(&tmp)
		s.LastSort = tmp
		delete(raw, "lastsort")
	}

	// 剩下的就是不固定的
	s.Extras = make(map[string]string)
	for k, v := range raw {
		var val string
		_ = json.Unmarshal(v, &val)
		s.Extras[k] = val
	}
	return nil
}
