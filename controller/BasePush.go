package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
	"github.com/sunvc/NoLets/push"
	"github.com/sunvc/apns2"
)

// BasePush 处理基础推送请求
// 验证推送参数并执行推送操作
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

		passwd := common.LocalConfig.System.PushPassword

		if len(passwd) > 0 && passwd == c.GetHeader("X-PUSH-PASSWD") {

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
		// 如果 title, subtitle 和 body 都为空，设置静默推送模式
		if result.PushType == 0 {
			return apns2.PushTypeBackground
		}
		return apns2.PushTypeAlert
	}()

	if err := push.BatchPush(result, pushType); err != nil {
		c.JSON(http.StatusOK, common.Failed(http.StatusInternalServerError, "push failed: %v", err))
		return
	}

	// 如果是管理员，加入到未推送列表
	if id, ok := result.Get(common.ID).(string); common.Admin(c) && ok && len(id) > 0 {
		UpdateNotPushedData(id, result, pushType)
	}

	c.JSON(http.StatusOK, common.Success())
}
