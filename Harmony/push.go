package Harmony

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/apns2"
)

func AutoPush(params *common.ParamsResult) error {

	var tokens []string

	for _, user := range params.Users {
		if user.OS == string(common.HARMONY) {
			tokens = append(tokens, user.Token)
		}
	}

	if len(tokens) == 0 {
		return errors.New("no token")
	}

	if params.PushType == apns2.PushTypeBackground {
		if tem := params.GetString(common.ID); tem != "" {
			return Delete(int(UUIDToInt31(tem)), tokens)
		}
		return errors.New("NOT ID")
	}
	return Push(params, tokens)
}

func Push(params *common.ParamsResult, tokens []string) error {
	ProjectId := common.LocalConfig.Harmony.ProjectID
	url := fmt.Sprintf("https://push-api.cloud.huawei.com/v3/%s/messages:send", ProjectId)
	// 构建推送请求体
	req := PushRequest{
		Payload: Payload{
			Notification: Notification{
				Category: "IM",
				Title:    "New order message",
				Body:     "Please handle it in time",
				ClickAction: ClickAction{
					ActionType: 0,
					Data: map[string]interface{}{
						string(common.ID):  params.GetString(common.ID),
						string(common.URL): params.GetString(common.URL),
					},
				},
			},
		},
		PushOptions: PushOptions{
			TestMessage: true,
			CollapseKey: -1,
		},
		Target: Target{
			Token: tokens,
		},
	}

	ID := params.GetString(common.ID)
	sound := params.GetString(common.SOUND)
	req.Payload.Notification.NotifyID = int(UUIDToInt31(ID))

	if sound != "" {
		req.Payload.Notification.Sound = normalizeSound(sound)
	}

	if avatar := params.GetString(common.ICON); avatar != "" {
		req.Payload.Notification.OverlayIcon = avatar
	}

	if png := params.GetString(common.IMAGE); png != "" {
		req.Payload.Notification.Image = png
	}

	if duration := params.GetString(common.CALL); duration != "" {
		if duraNumber, err := strconv.Atoi(duration); err == nil {
			if duraNumber >= 10 && duraNumber <= 60 {
				req.Payload.Notification.SoundDuration = duraNumber
				if sound == "" {
					req.Payload.Notification.Sound = "call.mp3"
				}
			}
		}
	}

	if data, err := sonic.Marshal(params.Params); err == nil {
		req.Payload.ExtraData = string(data)
		fmt.Println(string(data))
	} else {
		req.Payload.ExtraData = "***"
	}

	if common.LocalConfig.System.Debug {
		req.PushOptions = PushOptions{
			TestMessage: true,
		}
	}

	resp, err := sendPushMessage(url, 2, &req)

	if err != nil {
		return err
	}
	// 处理响应
	if resp.Code == "80000000" {
		return nil
	} else {
		return errors.New(fmt.Sprintf("推送失败: Code=%v, Msg=%v\n", resp.Code, resp.Msg))
	}

}

func Delete(notifyId int, tokens []string) error {
	ClientId := common.LocalConfig.Harmony.ClientId
	url := fmt.Sprintf("https://push-api.cloud.huawei.com/v1/%s/messages:revoke", ClientId)
	resp, err := sendPushMessage(url, 2, &DeleteRequest{
		NotifyId: notifyId,
		Token:    tokens,
	})

	if err != nil {
		return err
	}
	// 处理响应
	if resp.Code == "80000000" {
		return nil
	} else {
		return errors.New(fmt.Sprintf(" 删除失败: Code=%s, Msg=%s\n", resp.Code, resp.Msg))
	}
}

func sendPushMessage[T any](url string, pushType int, req *T) (*PushResponse, error) {

	accessToken, err := GetToken()

	if err != nil {
		return nil, err
	}

	// 序列化请求体
	body, err := sonic.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("push-type", fmt.Sprintf("%d", pushType))

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var pushResp PushResponse
	if err := sonic.Unmarshal(respBody, &pushResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 原始响应: %s", err, string(respBody))
	}

	return &pushResp, nil
}

func UUIDToInt31(uuid string) int32 {
	hash := sha256.Sum256([]byte(uuid))

	// 取前 4 个字节
	n := uint32(hash[0])<<24 |
		uint32(hash[1])<<16 |
		uint32(hash[2])<<8 |
		uint32(hash[3])

	return int32(n & 0x7fffffff)
}

func normalizeSound(sound string) string {
	sound = strings.TrimSuffix(sound, ".caf")

	if !strings.Contains(sound, ".") {
		sound += ".mp3"
	}

	return sound
}
