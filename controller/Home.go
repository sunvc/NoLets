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

// Home 处理首页请求
// 支持两种功能:
// 1. 通过id参数移除未推送数据
// 2. 生成二维码图片
func Home(c *gin.Context) {

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

	if data := c.GetHeader("X-DATA"); len(data) > 10 {
		ProxyDownload(c, data)
		return
	}

	if id := c.Query("id"); id != "" {
		NotPushedDataList.Delete(id)
		c.Status(http.StatusOK)
		return
	}

	url := common.GetClientHost(c)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"ICP":           common.LocalConfig.System.ICPInfo,
		"URL":           template.URL(url),
		"LOGORAW":       template.HTML(common.LOGORAW),
		"BACKGROUNDSVG": template.URL(common.LogoSvgImage("ff00000f", false)),
		"DOCS":          "https://wiki.wzs.app",
	})
}

func ProxyDownload(c *gin.Context, data string) {
	if !common.LocalConfig.System.ProxyDownload {
		c.String(http.StatusBadRequest, "missing")
		return
	}
	// 获取用户请求的下载地址
	targetURL, err := common.Decrypt(data, common.LocalConfig.System.SignKey)

	if err != nil {
		c.String(http.StatusBadRequest, "missing X-DATA header")
		return
	}
	ProxyDownloadData(c, targetURL)

}

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

	// 发起请求获取远程文件
	// 2. 创建带超时的HTTP客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0") // CDN 必须加 UA，否则降速

	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "fetch error: %v", err)
		return
	}
	defer resp.Body.Close()

	// 设置返回头，保持文件名或类型
	for k, v := range resp.Header {
		if len(v) > 0 {
			c.Writer.Header().Set(k, v[0])
		}
	}

	// 直接流式复制响应体给用户
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
