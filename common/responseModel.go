package common

import (
	"fmt"
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
	return BaseResp{
		Code:      200,
		Message:   "success",
		Data:      data,
		Timestamp: DateNow().Unix(),
		TraceID:   TraceID(c),
	}
}

// Failed creates a failed response with the specified error code and formatted message.
func Failed(c *gin.Context, code int, format string, args ...any) BaseResp {
	msg := func() string {
		if len(args) > 0 {
			return fmt.Sprintf(format, args...)
		}
		return format
	}()
	return BaseResp{
		Code:      code,
		Message:   msg,
		Data:      nil,
		Timestamp: DateNow().Unix(),
		TraceID:   TraceID(c),
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
