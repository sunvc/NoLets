package Harmony

import (
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sunvc/NoLets/common"
)

var (
	token      string
	expiration = time.Now()
)

func GetToken() (string, error) {
	var err error
	saKey := common.LocalConfig.Harmony

	fmt.Println(saKey)

	if saKey.KeyID == "" || saKey.SubAccount == "" || saKey.PrivateKey == "" {
		return "", errors.New("invalid service account key file: missing required fields")
	}

	if expiration.Before(time.Now()) {
		token, err = generateJWTToken(saKey)
		if err != nil {
			expiration = time.Now()
			token = ""
			return "", err
		}

		// 提前 5 分钟失效，避免刚好卡在 JWT 过期时间
		expiration = time.Now().Add(55 * time.Minute)
	}

	return token, nil
}

func generateJWTToken(saKey common.Harmony) (string, error) {

	formattedPrivateKey, err := formatPrivateKey(saKey.PrivateKey)
	if err != nil {
		return "", err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(formattedPrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	token, err := buildJWTToken(saKey.KeyID, saKey.TokenURI, saKey.SubAccount)
	if err != nil {
		return "", err
	}

	return token.SignedString(privateKey)
}

// buildJWTToken 构造 JWT token 对象
func buildJWTToken(keyID, aud, subAccount string) (*jwt.Token, error) {
	now := time.Now().UTC()
	iat := now.Unix()
	exp := iat + 3600 // token 过期时间：一小时后

	claims := jwt.MapClaims{
		// 实际开发时请将公网地址存储在配置文件或数据库
		"aud": aud,
		"iss": subAccount,
		"exp": exp,
		"iat": iat,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)

	// 设置 header
	token.Header["kid"] = keyID
	token.Header["typ"] = "JWT"
	token.Header["alg"] = "PS256"

	return token, nil
}

// formatPrivateKey 格式化私钥字符串为 PEM 格式
func formatPrivateKey(privateKeyStr string) (string, error) {
	trimmed := strings.TrimSpace(privateKeyStr)

	// 如果已经是 PEM 格式，则直接返回
	if strings.HasPrefix(trimmed, "-----BEGIN PRIVATE KEY-----") &&
		strings.HasSuffix(trimmed, "-----END PRIVATE KEY-----") {
		return trimmed, nil
	}

	block, _ := pem.Decode([]byte(trimmed))
	if block == nil {
		return "", errors.New("failed to decode PEM block")
	}

	pemBytes := pem.EncodeToMemory(block)
	if pemBytes == nil {
		return "", errors.New("failed to encode private key to PEM format")
	}

	return string(pemBytes), nil
}
