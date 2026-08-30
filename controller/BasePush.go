package controller

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
	"github.com/sunvc/NoLets/push"
	"github.com/sunvc/apns2"
)

// BasePush handles basic push requests.
// It validates push parameters and executes the push operation.
func BasePush(c *gin.Context) {

	result := common.NewParamsResult(c)

	if result.PushType == "0" {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Incorrect Format"))
		return
	}

	for _, key := range result.Keys {
		if len(key) > 5 {
			if user, err := database.DB.DeviceTokenByKey(key); err == nil {
				result.Users = append(result.Users, *user)
			}

		}
	}

	if name, ok := result.Params.Get(common.PUSHGROUPNAME); ok {
		if nameStr, bok := name.(string); bok {
			users, err := database.DB.DeviceTokenByGroup(nameStr)
			if err == nil && len(users) > 0 {
				for _, user := range users {
					result.Users = append(result.Users, *user)
				}
			}
		}
	}

	result.Users = common.UserUnique(result.Users)

	if !common.Admin(c) {
		if len(result.Users) > common.LocalConfig.System.MaxDeviceKeyArrLength {
			result.Users = result.Users[:common.LocalConfig.System.MaxDeviceKeyArrLength]
		}
	}

	if result.PushType == apns2.PushTypeLocation {

		if len(result.Users) <= 0 {
			c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "failed to get device token"))
			return
		}

		if err := push.LocationPush(result); len(err) > 0 {
			data, _ := sonic.Marshal(err)
			c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "failed to push location: %v", string(data)))
			return
		}

		c.JSON(http.StatusOK, common.Success(c, nil))
		return
	}

	if len(result.Users) <= 0 {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "failed to get device token"))
		return
	}

	if errs := push.BatchPush(result, result.PushType); len(errs) > 0 {
		data, _ := sonic.Marshal(errs)
		c.JSON(http.StatusOK, common.Failed(c, http.StatusInternalServerError, "push failed: %v", string(data)))
		return
	}

	c.JSON(http.StatusOK, common.Success(c, nil))
}
