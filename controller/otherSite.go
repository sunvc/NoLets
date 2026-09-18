package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
)

func AppleSite(c *gin.Context) {
	appID := fmt.Sprintf("%s.%s", common.LocalConfig.Apple.TeamID, common.LocalConfig.Apple.Topic)
	c.JSON(200, gin.H{
		"applinks": gin.H{
			"details": []gin.H{
				{
					"appID": appID,
					"paths": []string{"*"},
				},
			},
		},
	})
}

func HarmonyOS(c *gin.Context) {
	c.JSON(200, gin.H{
		"applinking": gin.H{
			"apps": []gin.H{
				{
					"appIdentifier": "6917590980754890055",
					"index":         1,
				},
			},
		},
	})
}

// RobotText handles robots.txt requests.
func RobotText(c *gin.Context) {
	c.String(http.StatusOK, "User-agent: * \nDisallow: / \nAllow: /$ \n")
}
