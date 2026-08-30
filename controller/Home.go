package controller

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/push"
)

// Home handles page requests.
// It supports two functions:
// 1. Remove unpunished data via id parameter
// 2. Generate QR code image
func Home(c *gin.Context) {

	if common.LocalConfig.System.HideHome {
		c.String(http.StatusOK, "OK")
		return
	}

	if strings.ToUpper(c.Request.Method) == "POST" {
		token, expiry := push.GetToken()
		c.JSON(http.StatusOK, gin.H{
			"code":      http.StatusOK,
			"data":      token,
			"expiry":    expiry,
			"timestamp": time.Now().Unix(),
		})
		return
	}

	ua := strings.ToLower(c.GetHeader("User-Agent"))

	if strings.Contains(ua, "curl") || strings.Contains(ua, "wget") {
		if !common.LocalConfig.System.ProxyDownload {
			c.String(http.StatusBadRequest, "missing")
			return
		}
		DownloadProject(c)
		return
	}

	url := common.GetClientHost(c)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"ICP":           common.LocalConfig.System.ICPInfo,
		"URL":           template.URL(fmt.Sprintf("pb://server?text=%s", url)),
		"LOGORAW":       template.HTML(common.LOGORAW),
		"BACKGROUNDSVG": template.URL(common.LogoSvgImage("ff00000f", false)),
		"DOCS":          "https://wiki.wzs.app",
	})
}

// DownloadProject handles project download requests.
// It supports three platforms: Windows, macOS, and Linux.
func DownloadProject(c *gin.Context) {
	goos := strings.ToLower(c.GetHeader("os"))
	var name string
	if strings.Contains(goos, "win") {
		name = "windows_amd64"
	} else if strings.Contains(goos, "mac") || strings.Contains(goos, "darwin") {
		if strings.Contains(goos, "arm") {
			name = "darwin_arm64"
		} else {
			name = "darwin_amd64"
		}
	} else {
		if strings.Contains(goos, "arm") {
			name = "linux_arm64"
		} else {
			name = "linux_amd64"
		}
	}

	url := fmt.Sprintf("https://github.com/sunvc/NoLets/releases/download/%s/NoLets_%s.tar.gz", common.LocalConfig.System.Version, name)
	ProxyDownloadData(c, url)

}

// ProxyDownloadData performs the actual HTTP request to fetch the file and streams the response back to the client.
func ProxyDownloadData(c *gin.Context, targetURL string) {

	if targetURL == "" {
		c.String(http.StatusBadRequest, "missing URL")
		return
	}

	var transport = &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     10 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	// Initiate request to fetch remote file
	// 2. Create HTTP client with timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0") // CDN requires UA, otherwise speed will be limited

	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		if len(v) > 0 {
			c.Writer.Header().Set(k, v[0])
		}
	}

	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
