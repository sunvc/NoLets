package router

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/controller"
)

func Verification() gin.HandlerFunc {

	return func(c *gin.Context) {
		requestID, _ := uuid.NewUUID()
		c.Set("trace_id", requestID)

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
		c.Next()
	}
}

// CheckDotParamMiddleware checks if the first path parameter of the GET request contains '.'
func CheckDotParamMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if value := c.Param("deviceKey"); strings.Contains(value, ".") {
			controller.GetImage(c)
			c.Abort()
			return
		}
		// Allow request
		c.Next()
	}
}

func GCMDecryptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512)

		userAgent := c.GetHeader(common.HeaderUserAgent)
		if !strings.HasPrefix(strings.ToLower(userAgent), strings.ToLower(common.APPNAME)) {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(http.StatusUnauthorized, "SB"))
			return
		}

		if len(common.LocalConfig.System.SignKey) < 10 {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")

		if sign := c.GetHeader("X-Signature"); sign != "" {
			header = sign
		}

		if header == "" {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				http.StatusUnauthorized,
				"missing signature",
			))
			return
		}

		timestampStr, err := common.Decrypt(header, common.LocalConfig.System.SignKey)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature verification failed！err1:", err)
			return
		}

		timestamp, err := strconv.ParseFloat(timestampStr, 64)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature verification failed！err2:", err)
			return
		}

		now := time.Now().Unix()
		if now-int64(timestamp) > 10 || now < int64(timestamp) {
			c.AbortWithStatusJSON(http.StatusOK, common.Failed(
				http.StatusUnauthorized,
				"missing signature",
			))
			log.Println("Signature verification failed！timestamp:", timestampStr)
			return
		}

		log.Println("Signature verification successful！")
		// Decryption successful, save to context
		c.Set("decrypted", timestamp)
		c.Next()

	}
}
