package api_helper

import (
	"120bid/types"
)

// Response 通用响应
type Response struct {
	Status   int     `json:"status"`
	Message  string  `json:"msg"`
	ErrCode  int     `json:"errCode"`
	Count    int     `json:"count"`
	LastSort string  `json:"lastSort"`
	Items    []*Data `json:"items"`
}
type Data struct {
	Url    string `json:"url" gorm:"comment:原链接"`
	Title  string `json:"title" gorm:"comment:标题"`
	Status string `json:"status" gorm:"comment:类型"`
	Area   string `json:"area" gorm:"comment:地区"`
	City   string `json:"city" gorm:"comment:城市"`
	User   string `json:"user" gorm:"comment:采购方"`
	Date   string `json:"date" gorm:"comment:日期"`
	Html   string `json:"html" gorm:"comment:html内容"`
}

func (d *Data) Find() *Data {
	var data Data
	err := types.DB.Where("Title = ?", d.Title).First(&data).Error
	if err != nil {
		return nil
	}
	return &data
}

func (d *Data) Create() {
	types.DB.Create(d)
}
