package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
)

// Register handles device registration requests.
// It allows clients to upload a DEVICETOKEN and bind it to a specific DEVICEKEY.
// If the Key does not exist, a new one will be generated.
func Register(c *gin.Context) {
	var err error
	var device common.DeviceInfo

	if err = c.BindJSON(&device); err != nil {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "%v", err))
		return
	}

	if len(strings.TrimSpace(device.Token)) > 200 {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Invalid deviceToken"))
		return
	}
	device.Key, err = database.DB.SaveDeviceTokenByKey(common.User{
		Key:      device.Key,
		Token:    device.Token,
		Talk:     device.Talk,
		Location: device.Location,
		Group:    device.Group,
	})

	if err != nil {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusInternalServerError, "device registration failed: %v", err))
		return
	}

	if common.LocalConfig.System.Voice {
		device.Core = 2
	}

	c.JSON(http.StatusOK, common.Success(c, device))
}

// Restore restores or checks a device Key.
// It validates if a device Key is active, or restores/creates an empty device Key under admin privileges.
func Restore(c *gin.Context) {

	deviceKey := c.Param("deviceKey")

	if !common.Admin(c) && len(deviceKey) != 22 {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Invalid deviceKey"))
		return
	}

	if deviceKey == "" {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "device key is empty"))
		return
	}

	if database.DB.KeyExists(deviceKey) {
		c.JSON(http.StatusOK, common.Success(c, nil))
		return
	} else {
		_, err := database.DB.SaveDeviceTokenByKey(common.User{Key: deviceKey})
		if err != nil {
			c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "key save err"))
			return
		}
		c.JSON(http.StatusOK, common.Success(c, nil))
	}

}
