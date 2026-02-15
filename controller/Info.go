package controller

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
	"github.com/sunvc/NoLets/serverInfo"
)

func Info(c *gin.Context) {

	mode := c.Query("mode")
	admin := Verification(c)

	if admin && mode == "monitor" {
		c.JSON(http.StatusOK, serverInfo.FetchData())
		return
	}

	if admin && mode == "processes" {
		c.JSON(http.StatusOK, serverInfo.FetchProcesses())
		return
	}

	system := common.LocalConfig.System

	results := gin.H{
		"version": system.Version,
		"build":   system.BuildDate,
		"commit":  system.CommitID,
	}

	if admin {
		devices, _ := database.DB.CountAll()
		results["devices"] = devices
		results["arch"] = runtime.GOOS + "/" + runtime.GOARCH
		results["cpu"] = runtime.NumCPU()
	}
	c.JSON(http.StatusOK, results)
}

// Ping handles heartbeat requests.
// Returns current server status.
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, common.BaseResp{
		Code:      http.StatusOK,
		Message:   "pong",
		Timestamp: time.Now().Unix(),
	})
}

// Health handles health check requests.
func Health(c *gin.Context) { c.String(http.StatusOK, "OK") }

// Verification checks if the user has admin privileges.
func Verification(c *gin.Context) bool {
	admin, ok := c.Get("admin")
	return ok && admin.(bool)
}
