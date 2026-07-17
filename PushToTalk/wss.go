package PushToTalk

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// 配置 Upgrader，用于将 HTTP 协议升级为 WebSocket 协议
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 生产环境下建议对 Origin 进行安全校验，这里为了方便测试直接允许所有跨域
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
