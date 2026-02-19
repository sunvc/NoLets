package controller

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
)

// GetImage handles image retrieval requests for logos and icons.
func GetImage(c *gin.Context) {
	fileName := c.Param("deviceKey")
	color := c.Query("color")

	if fileName == "logo.svg" {
		c.Data(http.StatusOK, common.MIMEImageSvg, []byte(common.LogoSvgImage(color, true)))
		return
	}

	if strings.HasSuffix(fileName, "ico") || strings.HasSuffix(fileName, "png") {
		if strings.HasPrefix(fileName, "og") {
			fileName = "og.png"
		} else {
			fileName = "logo.png"
		}
	}

	path := filepath.Join("static", fileName)

	data, err := common.StaticFS.ReadFile(path)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Data(http.StatusOK, common.MIMEImagePng, data)
}
