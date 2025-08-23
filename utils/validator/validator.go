package validator

import "time"

// IsDateValid 检查日期字符串是否符合 "2006-01-02" 格式
func IsDateValid(dateStr string) bool {
	if dateStr == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
