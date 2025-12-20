package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
)

// Register 处理设备注册请求
// 支持 GET 和 POST 两种请求方式:
// GET: 检查设备key是否存在
// POST: 注册新的设备token
func Register(c *gin.Context) {

	if c.Request.Method == "GET" {
		deviceKey := c.Param("deviceKey")
		if deviceKey == "" {
			c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "device key is empty"))
			return
		}
		if database.DB.KeyExists(deviceKey) {
			c.JSON(http.StatusOK, common.Success())
			return
		} else {
			admin := Verification(c)

			if admin {
				_, err := database.DB.SaveDeviceTokenByKey(deviceKey, "", "")
				if err != nil {
					c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "device key is not exist"))
					return
				}
				c.JSON(http.StatusOK, common.Success())
				return
			}

			c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "device key is not exist"))
		}
		return
	}

	var err error
	var device common.DeviceInfo

	if err = c.BindJSON(&device); err != nil {
		c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "failed to get device token: %v", err))
		return
	}

	if len(device.Token) > 128 {
		c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "Invalid deviceToken"))
		return
	}
	log.Println(device)
	device.Key, err = database.DB.SaveDeviceTokenByKey(device.Key, device.Token, device.Group)

	if err != nil {
		c.JSON(http.StatusOK, common.Failed(http.StatusInternalServerError, "device registration failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, common.Success(device))
}
