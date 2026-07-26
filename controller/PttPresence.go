package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	PushToTalk2 "github.com/sunvc/NoLets/controller/PushToTalk"
)

// PttPresence 客户端周期性上报自身位置。仅广播 update 事件,不做频道拓扑变更。
// 请求体沿用 PttUser + channels(可选,兼容 JoinParams 打通签名逻辑)。
func PttPresence(c *gin.Context) {
	var req PushToTalk2.JoinParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, common.Failed(c, -1, err.Error(), nil))
		return
	}
	if req.ID == "" {
		c.JSON(200, common.Failed(c, -1, "id required", nil))
		return
	}
	PushToTalk2.BroadcastUpdate(req.PttUser)
	c.JSON(200, common.Success(c, 0))
}
