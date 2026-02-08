package query

import "errors"

var (
	// 执行控制类错误
	ErrTimeout = errors.New("query step timeout")

	// 调度 / 配置类错误
	ErrPlanNotFound = errors.New("query plan not found")

	ErrInvalidItem = errors.New("invalid item")

	ErrHashNotFound = errors.New("hash not found")
)
