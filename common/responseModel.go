package common

import (
	"fmt"
	"time"
)

// BaseResp represents the standard JSON response structure for the API.
type BaseResp struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Success creates a successful response (HTTP 200) with optional data.
func Success(data ...interface{}) BaseResp {
	var result interface{}

	if len(data) > 0 {
		result = data[0]
	}
	return BaseResp{
		Code:      200,
		Message:   "success",
		Data:      result,
		Timestamp: DateNow().Unix(),
	}
}

// Failed creates a failed response with the specified error code and formatted message.
func Failed(code int, message string, args ...interface{}) BaseResp {
	return BaseResp{
		Code:      code,
		Message:   fmt.Sprintf(message, args...),
		Timestamp: DateNow().Unix(),
	}
}

// BaseRes creates a custom response with the specified status code, message, and optional data.
func BaseRes(code int, message string, data ...interface{}) BaseResp {
	var result interface{}

	if len(data) > 0 {
		result = data[0]
	}

	return BaseResp{
		Code:      code,
		Message:   message,
		Data:      result,
		Timestamp: DateNow().Unix(),
	}
}

type DeviceInfo struct {
	Key   string `json:"key"`
	Token string `json:"token"`
	Group string `json:"group,omitempty"`
}

// DateNow returns the current time in UTC.
func DateNow() time.Time {
	return time.Now().UTC()
}
