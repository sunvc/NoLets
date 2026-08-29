package common

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v4"
	"github.com/sunvc/apns2"
)

type NotPushedData struct {
	ID           string          `json:"id"`
	CreateDate   time.Time       `json:"createDate"`
	LastPushDate time.Time       `json:"lastPushDate"`
	Count        int             `json:"count"`
	Params       *ParamsResult   `json:"params"`
	PushType     apns2.EPushType `json:"pushType"`
}

func BaseDir(path ...string) string {
	dataDir := LocalConfig.System.DataDir
	if len(path) == 0 {
		return dataDir
	}
	return filepath.Join(append([]string{dataDir}, path...)...)
}

func Unique[T comparable](list []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(list))

	for _, v := range list {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Contains checks if the slice contains the specified element
func Contains[T comparable](slice []T, val T) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func Admin(ctx *gin.Context) bool {
	admin, ok := ctx.Get("admin")
	if ok {
		auth, success := admin.(bool)
		return success && auth
	}
	return false
}

func TraceID(ctx *gin.Context) string {
	admin, ok := ctx.Get("trace_id")
	if ok {
		auth, success := admin.(string)
		if success {
			return auth
		}
	}
	return ""
}

// GetClientHost 安全且健壮地获取客户端访问的完整 Base URL (例如: https://example.com)
func GetClientHost(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	scheme := getClientScheme(c)
	host := getClientHostHeader(c)

	return fmt.Sprintf("%s://%s", scheme, host)
}

// 获取协议类型 (http / https)
func getClientScheme(c *gin.Context) string {
	// 1. 优先从标准/常用 Header 获取
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		// 多层代理时 Header 格式可能为: "https, http"，取第一个
		if idx := strings.IndexByte(proto, ','); idx != -1 {
			proto = proto[:idx]
		}
		proto = strings.TrimSpace(strings.ToLower(proto))
		if proto == "http" || proto == "https" {
			return proto
		}
	}

	// 2. 检查特定网关/云厂商的协议 Header (如 Cloudflare, AWS 等)
	if cfVisitor := c.GetHeader("CF-Visitor"); cfVisitor != "" {
		if strings.Contains(cfVisitor, `"scheme":"https"`) {
			return "https"
		}
	}

	// 3. 检查常规 TLS 连接
	if c.Request.TLS != nil {
		return "https"
	}

	return "http"
}

// 获取真实 Host (域名 + 可选端口)
func getClientHostHeader(c *gin.Context) string {
	var host string

	// 1. 优先尝试 X-Forwarded-Host (反向代理保留的原域名)
	if fHost := c.GetHeader("X-Forwarded-Host"); fHost != "" {
		// 同理，多层代理可能返回 "example.com, internal-proxy.com"
		if idx := strings.IndexByte(fHost, ','); idx != -1 {
			fHost = fHost[:idx]
		}
		host = strings.TrimSpace(fHost)
	}

	// 2. 回退到 Request.Host
	if host == "" {
		host = c.Request.Host
	}

	// 3. 兜底回退到 URL.Host 或 Header 中的 Host
	if host == "" && c.Request.URL != nil {
		host = c.Request.URL.Host
	}

	// 清理多余空格及末尾斜杠，防止非法 Inject
	host = strings.TrimRight(strings.TrimSpace(host), "/")

	// 4. 防御极罕见的空 Host 情况
	if host == "" {
		host = "localhost"
	}

	return host
}

func IsFileInDirectory(dirPath, fileName string) (bool, error) {
	// Normalize the directory path
	dirPath = filepath.Clean(dirPath)

	// Check if the directory exists
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // If the directory does not exist, return that the file is not in the directory
		}
		return false, fmt.Errorf("Error checking directory status: %w", err)
	}

	// Confirm that the path points to a directory
	if !dirInfo.IsDir() {
		return false, fmt.Errorf("Path %q is not a directory", dirPath)
	}

	// Build the full path of the file
	filePath := filepath.Join(dirPath, fileName)

	// Check if the file exists
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File does not exist
		}
		return false, fmt.Errorf("Error checking file status: %w", err)
	}

	return true, nil
}

func FilterShortStrings(input []string, minNumber, maxNumber int) []string {
	result := make(map[string]struct{})

	for _, s := range input {
		if len(s) >= minNumber && len(s) <= maxNumber {
			result[s] = struct{}{}
		}
	}

	output := make([]string, 0, len(result))
	for s := range result {
		output = append(output, s)
	}

	return output
}

func UserID(name ...string) string {
	return shortuuid.NewWithNamespace(strings.Join(name, ""))
}

func Decrypt(signText string, key string) (string, error) {

	// Base64 URL Safe -> Standard Base64
	signText = strings.ReplaceAll(signText, "-", "+")
	signText = strings.ReplaceAll(signText, "_", "/")
	if m := len(signText) % 4; m != 0 {
		signText += strings.Repeat("=", 4-m)
	}

	data, err := base64.StdEncoding.DecodeString(signText)
	if err != nil {
		return "", errors.New("missing signature")
	}

	nonceSize := 12
	tagSize := 16
	if len(data) <= nonceSize+tagSize {
		return "", errors.New("missing signature")
	}
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize : len(data)-tagSize]
	tag := data[len(data)-tagSize:]

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", errors.New("missing signature")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("missing signature")
	}

	// CryptoKit puts the tag separately at the end, need to append to ciphertext
	decrypted, err := aesgcm.Open(nil, nonce, append(ciphertext, tag...), nil)
	if err != nil {

		return "", errors.New("missing signature")
	}

	return string(decrypted), nil
}
