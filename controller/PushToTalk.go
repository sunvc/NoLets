package controller

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sunvc/NoLets/PushToTalk"
	"github.com/sunvc/NoLets/common"
)

// PushTask 代表发往单个用户的推送任务（第二级队列使用）

func PttConnect(c *gin.Context) {

	var req PushToTalk.JoinParams

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, common.Failed(c, -1, err.Error()))
		return
	}

	if req.Token == "" {
		PushToTalk.SyncChannels(req.PttUser, []string{})
		c.JSON(200, common.Success(c, []PushToTalk.JoinResponse{}))

	} else {
		var response []PushToTalk.JoinResponse

		PushToTalk.SyncChannels(req.PttUser, req.Channels)

		for _, channel := range req.Channels {
			ch, ok := PushToTalk.Channels[channel]
			if !ok {
				continue
			}
			response = append(response, PushToTalk.JoinResponse{
				Channel: channel,
				Host:    req.Host,
				Users:   ch.UserList(),
			})
		}

		c.JSON(200, common.Success(c, response))
	}
	
}

func PttVoice(c *gin.Context) {

	if c.Request.Method == "POST" {

		fileName := c.GetHeader("X-PFA")

		if fileName == "" {
			c.JSON(200, common.Failed(c, -1, "upload file error"))
			return
		}

		id, channel, timestamp := getUserData(fileName)

		if !veryPttTimestamp(timestamp, 60) {
			c.JSON(200, common.Failed(c, -1, "Error Data!"))
			return
		}

		savePath := filepath.Join("data", "voices", fileName)

		file, err := os.Create(savePath)
		defer func() { _ = file.Close() }()

		if err != nil {
			c.JSON(200, common.Failed(c, -1, err.Error()))
			return
		}

		_, err = io.Copy(file, c.Request.Body)

		if err != nil {
			c.JSON(200, common.Failed(c, -1, err.Error()))
			return
		}

		msg := PushToTalk.VoiceMessage{
			ID:        uuid.New().String(),
			Channel:   channel,
			FileName:  fileName,
			Sender:    id,
			Timestamp: timestamp,
			CreatedAt: time.Now().UnixMilli(),
		}

		PushToTalk.MsgQueue <- msg

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
		fmt.Println(path)
		if _, err := os.Stat(path); err != nil {
			c.AbortWithStatus(404)
			return
		}

		c.Header("Content-Type", "audio/ogg")
		c.File(path)
		return
	}

	c.JSON(200, common.Failed(c, -1, "Error Method"))

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
