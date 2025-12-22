package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/serverInfo"
)

// GetServerInfo returns server monitoring information.
func GetServerInfo(c *gin.Context) {

	if Verification(c) {
		data, err := serverInfo.GetServerInfo()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		c.Data(http.StatusOK, common.MIMEApplicationJSON, data)
	} else {
		c.JSON(http.StatusOK, common.Failed(200, "No Permission!"))
	}

}
