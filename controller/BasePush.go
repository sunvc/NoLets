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

	if result.PushType == -1 {
		if len(result.Keys) > 0 {
			deviceKey := result.Keys[0]
			token, err := database.DB.DeviceTokenByKey(deviceKey)
			if err != nil {
				c.JSON(http.StatusOK, common.Failed(http.StatusInternalServerError, "failed to get device token: %v", err))
				return
			}
			c.JSON(http.StatusOK, common.Success(token))
			return
		}
		c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "Incorrect Format"))
		return
	}

	if len(result.Tokens) <= 0 {
		for _, key := range result.Keys {
			if len(key) > 5 {
				if token, err := database.DB.DeviceTokenByKey(key); err == nil {
					result.Tokens = append(result.Tokens, token)
				}

			}
		}
		result.Tokens = common.Unique(result.Tokens)

		if common.Admin(c) {

			if name, ok := result.Params.Get(common.PushGroupName); ok {
				if nameStr, bok := name.(string); bok {
					tokens, err := database.DB.DeviceTokenByGroup(nameStr)
					tokens = common.Unique(tokens)
					if err == nil && len(tokens) > 0 {
						result.Tokens = append(result.Tokens, tokens...)
					}
				}

			}
		}
	}

	if len(result.Tokens) <= 0 {
		c.JSON(http.StatusOK, common.Failed(http.StatusBadRequest, "Failed to get device token"))
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
		c.JSON(http.StatusOK, common.Failed(http.StatusInternalServerError, "push failed: %v", err))
		return
	}

	// If admin, add to not-pushed list
	if id, ok := result.Get(common.ID).(string); common.Admin(c) && ok && len(id) > 0 {
		UpdateNotPushedData(id, result, pushType)
	}

	c.JSON(http.StatusOK, common.Success())
}
