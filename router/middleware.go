package router

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/controller"
)

func Verification() gin.HandlerFunc {

	return func(c *gin.Context) {

		device := c.GetHeader("X-Device")
		if device != "" && common.Contains[string](common.LocalConfig.System.Auths, device) {
			c.Set("admin", true)
			return
		}

		localUser := common.LocalConfig.System.User
		localPassword := common.LocalConfig.System.Password
		// Configured account password, perform identity verification
		if localUser != "" && localPassword != "" {
			// Prioritize Basic Auth
			user, pass, hasAuth := c.Request.BasicAuth()
			if !hasAuth {
				// If no Basic Auth, try to get from query parameters
				user = c.Query(common.UserName)
				pass = c.Query(common.Password)

				if c.Request.Method == http.MethodPost {
					if user == "" {
						user = c.PostForm(common.UserName)
					}
					if pass == "" {
						pass = c.PostForm(common.Password)
					}
				}
			}

			if user == localUser && pass == localPassword {
				c.Set("admin", true)
				return
			}

		}

		// If no authentication information
		c.Set("admin", false)
	}
}

// CheckDotParamMiddleware checks if the first path parameter of the GET request contains '.'
func CheckDotParamMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if value := c.Param("deviceKey"); strings.Contains(value, ".") {
			controller.GetImage(c)
			c.Abort()
		}
	}
}

func GCMDecryptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if common.Admin(c) {
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512)

		userAgent := c.GetHeader(common.HeaderUserAgent)
		if !strings.HasPrefix(strings.ToLower(userAgent), strings.ToLower(common.APPNAME)) {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(c, http.StatusUnauthorized, "SB"))
			return
		}

		if len(common.LocalConfig.System.SignKey) < 10 {
			return
		}
		header := c.GetHeader("Authorization")

		if sign := c.GetHeader("X-Signature"); sign != "" {
			header = sign
		}

		if header == "" {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				c,
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("missing signature")
			return
		}

		timestampStr, err := common.Decrypt(header, common.LocalConfig.System.SignKey)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				c,
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature failed！err1:", err)
			return
		}

		timestamp, err := strconv.ParseFloat(timestampStr, 64)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				c,
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature failed！err2:", err)
			return
		}

		now := time.Now().Unix()
		if now-int64(timestamp) > 10 || now < int64(timestamp) {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				c,
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature failed！timestamp:", timestampStr)
			return
		}
	}
}

// GinRecovery 替换 Gin 默认 Recovery 的中间件，用于捕获 Panic
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				// 转换为 error 类型以便使用 errors 包
				if ne, ok := err.(error); ok {
					var opErr *net.OpError
					// 递归检查错误链中是否包含 *net.OpError
					if errors.As(ne, &opErr) {
						var sysErr *os.SyscallError
						if errors.As(opErr.Err, &sysErr) {
							se := sysErr.Syscall
							if se == "write" || se == "accept" {
								brokenPipe = true
							}
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					log.Println(c.Request.URL.Path, err, string(httpRequest))
					_ = c.Error(err.(error))
					c.Abort()
					return
				}
				log.Println("[Recovery from panic]", err, string(httpRequest), string(debug.Stack()))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
