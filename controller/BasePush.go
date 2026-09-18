package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/Harmony"
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

	var harmonyErr error
	var appleError error

	if users := result.GetUser(common.HARMONY); len(users) > 0 {
		harmonyErr = Harmony.AutoPush(result)
	}

	if users := result.GetUser(common.APPLE); len(users) > 0 {
		if result.PushType == apns2.PushTypeLocation {
			if err := push.LocationPush(result); len(err) > 0 {
				data, _ := sonic.Marshal(err)
				appleError = errors.New(fmt.Sprintf("failed to push location: %v", string(data)))
			}
		} else {
			if errs := push.BatchPush(result, result.PushType); len(errs) > 0 {
				data, _ := sonic.Marshal(errs)
				appleError = errors.New(fmt.Sprintf("push failed: %v", string(data)))
			}
		}
	}

	if harmonyErr != nil && appleError != nil {
		c.JSON(http.StatusOK,
			common.Failed(
				c,
				200,
				"harmony: %v; apple: %v", harmonyErr.Error(), appleError.Error(),
			),
		)
		return
	} else if harmonyErr != nil {
		c.JSON(http.StatusOK, common.Success(c, harmonyErr.Error()))
		return
	} else if appleError != nil {
		c.JSON(http.StatusOK, common.Success(c, appleError.Error()))
		return
	}

	c.JSON(http.StatusOK, common.Success(c, true))

}
