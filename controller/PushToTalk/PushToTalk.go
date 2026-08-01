package PushToTalk

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sunvc/NoLets/common"
)

func PttConnect(c *gin.Context) {
	var req JoinParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, common.Failed(c, -1, err.Error(), nil))
		return
	}
	if req.ID == "" {
		c.JSON(200, common.Failed(c, -1, "id required", nil))
		return
	}

	if req.Token == "" {
		old, ok := UserChannels[req.ID]
		if ok {
			drop := make(map[string]struct{}, len(req.Channels))
			for _, ch := range req.Channels {
				drop[ch] = struct{}{}
			}
			remaining := make([]string, 0, len(old))
			for ch := range old {
				if _, isDrop := drop[ch]; !isDrop {
					remaining = append(remaining, ch)
				}
			}
			SyncChannels(req.PttUser, remaining)
		}
		c.JSON(200, common.Success(c, []JoinResponse{}))
		return
	}

	req.PttUser.Timestamp = time.Now().UnixMilli()
	SyncChannels(req.PttUser, req.Channels)

	var response []JoinResponse
	ChannelLock.RLock()
	for _, channel := range req.Channels {
		ch, ok := Channels[channel]
		if !ok {
			continue
		}
		response = append(response, JoinResponse{
			Channel: channel,
			Host:    req.Host,
			Users:   ch.UserListResp(),
		})
	}
	ChannelLock.RUnlock()

	c.JSON(200, common.Success(c, response))
}

func PttVoice(c *gin.Context) {

	if c.Request.Method == "POST" {

		fileName := c.GetHeader("X-PFA")

		if fileName == "" {
			c.JSON(200, common.Failed(c, -1, "upload file error", nil))
			return
		}

		id, channel, timestamp := getUserData(fileName)

		if !veryPttTimestamp(timestamp, 60) {
			c.JSON(200, common.Failed(c, -1, "Error DATA!", nil))
			return
		}

		savePath := filepath.Join("data", "voices", fileName)

		file, err := os.Create(savePath)
		defer func() { _ = file.Close() }()

		if err != nil {
			c.JSON(200, common.Failed(c, -1, err.Error(), nil))
			return
		}

		_, err = io.Copy(file, c.Request.Body)

		if err != nil {
			c.JSON(200, common.Failed(c, -1, err.Error(), nil))
			return
		}

		msg := VoiceMessage{
			ID:        uuid.New().String(),
			Host:      common.GetClientHost(c),
			Channel:   channel,
			FileName:  fileName,
			Sender:    id,
			Timestamp: timestamp,
			CreatedAt: time.Now().UnixMilli(),
		}

		MsgQueue <- msg

		c.JSON(200, common.Success(c, 0))
		return
	}

	if c.Request.Method == "GET" {
		fileName := c.Param("name")

		if fileName == "" || len(fileName) < 6 {
			c.AbortWithStatus(404)
			return
		}

		path := common.BaseDir("voices", fileName)

		if _, err := os.Stat(path); err != nil {
			c.AbortWithStatus(404)
			return
		}

		c.Header("Content-Type", "audio/ogg")
		c.File(path)
		return
	}

	c.JSON(200, common.Failed(c, -1, "Error Method", nil))

}

func getUserData(fileName string) (id string, channel string, timestamp string) {
	ext := filepath.Ext(fileName) // ".ogg"
	name := fileName[:len(fileName)-len(ext)]
	parts := strings.Split(name, "-")
	if len(parts) != 4 {
		return "", "", ""
	}
	return parts[2], parts[1], parts[3]
}

func veryPttTimestamp(timestamp string, ext int) bool {
	// 32进制转回整数时间戳
	ts, err := strconv.ParseInt(timestamp, 32, 64)
	if err != nil {
		return false
	}
	seconds := time.Now().Sub(time.UnixMilli(ts)).Seconds()
	return seconds < float64(ext)
}
