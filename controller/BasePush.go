package controller

import (
	"net/http"

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

	if result.PushType == 2 {
		if len(result.Users) <= 0 {
			for _, key := range result.Keys {
				if len(key) > 5 {
					if user, err := database.DB.DeviceTokenByKey(key); err == nil {
						result.Users = append(result.Users, *user)
					}

				}
			}
		}
		result.Users = common.UserUnique(result.Users)

		if len(result.Users) <= 0 {
			c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Failed to get device token", nil))
			return
		}

		if err := push.LocationPush(result); len(err) > 0 {
			c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Failed to push location: ", err))
			return
		}

		c.JSON(http.StatusOK, common.Success(c, nil))
		return
	}

	if result.PushType == -1 {
		if len(result.Keys) > 0 {
			deviceKey := result.Keys[0]
			token, err := database.DB.DeviceTokenByKey(deviceKey)
			if err != nil {
				c.JSON(http.StatusOK, common.Failed(c, http.StatusInternalServerError, "failed to get device token: %v", err))
				return
			}
			c.JSON(http.StatusOK, common.Success(c, token))
			return
		}
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Incorrect Format", nil))
		return
	}

	if len(result.Users) <= 0 {
		for _, key := range result.Keys {
			if len(key) > 5 {
				if user, err := database.DB.DeviceTokenByKey(key); err == nil {
					result.Users = append(result.Users, *user)
				}

			}
		}
		result.Users = common.UserUnique(result.Users)

		if common.Admin(c) {

			if name, ok := result.Params.Get(common.PUSHGROUPNAME); ok {
				if nameStr, bok := name.(string); bok {
					users, err := database.DB.DeviceTokenByGroup(nameStr)
					var tokens []common.User
					for _, user := range users {
						tokens = append(tokens, *user)
					}
					tokens = common.UserUnique(tokens)
					if err == nil && len(tokens) > 0 {
						result.Users = append(result.Users, tokens...)
					}
				}

			}
		}
	}

	if len(result.Users) <= 0 {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusBadRequest, "Failed to get device token", nil))
		return
	}

	pushType := func() apns2.EPushType {
		// If title, subtitle, and body are all empty, set silent push mode
		if result.PushType == 0 {
			return apns2.PushTypeBackground
		}
		return apns2.PushTypeAlert
	}()

	if err := push.BatchPush(result, pushType); err != nil {
		c.JSON(http.StatusOK, common.Failed(c, http.StatusInternalServerError, "push failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, common.Success(c, nil))
}
