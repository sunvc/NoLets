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
		c.JSON(http.StatusOK, serverInfo.FetchData())
	} else {
		c.JSON(http.StatusOK, common.Failed(200, "No Permission!"))
	}

}

// GetProcesses returns server monitoring information.
func GetProcesses(c *gin.Context) {

	if Verification(c) {
		c.JSON(http.StatusOK, serverInfo.FetchProcesses())
	} else {
		c.JSON(http.StatusOK, common.Failed(200, "No Permission!"))
	}

}
