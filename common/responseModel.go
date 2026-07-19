package common

import (
	"time"

	"github.com/gin-gonic/gin"
)

// BaseResp represents the standard JSON response structure for the API.
type BaseResp struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	TraceID   string      `json:"trace,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Success creates a successful response (HTTP 200) with optional data.
func Success(c *gin.Context, data interface{}) BaseResp {
	return BaseRes(TraceID(c), 200, "success", data)
}

// Failed creates a failed response with the specified error code and formatted message.
func Failed(c *gin.Context, code int, message string, args interface{}) BaseResp {
	return BaseRes(TraceID(c), code, message, args)
}

// BaseRes creates a custom response with the specified status code, message, and optional data.
func BaseRes(id string, code int, message string, data interface{}) BaseResp {

	return BaseResp{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: DateNow().Unix(),
		TraceID:   id,
	}
}

type DeviceInfo struct {
	Key      string `json:"key"`
	Token    string `json:"token"`
	Talk     string `json:"talk,omitempty"`
	Location string `json:"location,omitempty"`
	Group    string `json:"group,omitempty"`
	Core     int64  `json:"core,omitempty"`
}

// DateNow returns the current time in UTC.
func DateNow() time.Time {
	return time.Now().UTC()
}
