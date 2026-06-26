package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
)

// Register handles device registration requests.
// It allows clients to upload a DeviceToken and bind it to a specific DeviceKey.
// If the Key does not exist, a new one will be generated.
func Register(c *gin.Context) {
	var err error
	var device common.DeviceInfo

	if err = c.BindJSON(&device); err != nil {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "failed to get device token: %v", err))
		return
	}

	if len(strings.TrimSpace(device.Token)) > 128 {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Invalid deviceToken"))
		return
	}
	device.Key, err = database.DB.SaveDeviceTokenByKey(device.Key, device.Token, device.Group)

	if err != nil {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusInternalServerError, "device registration failed: %v", err))
		return
	}

	device.Core = 2

	c.JSON(http.StatusOK, common.Success(c, device))
}

// Restore restores or checks a device Key.
// It validates if a device Key is active, or restores/creates an empty device Key under admin privileges.
func Restore(c *gin.Context) {
	deviceKey := c.Param("deviceKey")

	if deviceKey == "" {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "device key is empty"))
		return
	}

	if database.DB.KeyExists(deviceKey) {
		c.JSON(http.StatusOK, common.Success(c))
		return
	} else {
		if common.Admin(c) {
			_, err := database.DB.SaveDeviceTokenByKey(deviceKey, "", "")
			if err == nil {
				c.JSON(http.StatusOK, common.Success(c))
				return
			}
		}
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "device key is not exist"))
	}

}
