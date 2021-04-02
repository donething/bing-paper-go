package utils

import (
	"bing-paper-go/logger"
)

// 出错时，强制关闭程序
func Fatal(err error) {
	if err != nil {
		logger.Error.Fatal(err)
	}
}
