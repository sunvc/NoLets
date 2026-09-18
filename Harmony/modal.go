package Harmony

// ==================== 请求体结构定义 ====================

// ClickAction 点击动作
type ClickAction struct {
	ActionType int                    `json:"actionType"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// Notification 通知内容
type Notification struct {
	Category      string      `json:"category,omitempty"`      // 消息分类：MARKETING / NEWS / ...
	Title         string      `json:"title"`                   // 通知标题
	Body          string      `json:"body"`                    // 通知内容
	ProfileID     string      `json:"profileId,omitempty"`     // 通知通道ID
	Sound         string      `json:"sound,omitempty"`         // 铃声
	SoundDuration int         `json:"soundDuration,omitempty"` // 铃声持续时长
	OverlayIcon   string      `json:"overlayIcon,omitempty"`
	Image         string      `json:"image,omitempty"`
	ClickAction   ClickAction `json:"clickAction"` // 点击动作
	NotifyID      int         `json:"notifyId,omitempty"`
}

// Payload 消息载荷
type Payload struct {
	Notification Notification `json:"notification"`
	ExtraData    interface{}  `json:"extraData,omitempty"`
}

// Target 推送目标
type Target struct {
	Token []string `json:"token"` // Push Token 列表，单次最多1000个
}

// PushOptions 推送选项
type PushOptions struct {
	TestMessage bool   `json:"testMessage,omitempty"` // 测试消息标识
	TTL         int    `json:"ttl,omitempty"`         // 消息缓存时间（秒）
	BiTag       string `json:"biTag,omitempty"`       // 批量任务消息标识
	CollapseKey int    `json:"collapseKey,omitempty"`
}

// PushRequest 推送请求体
type PushRequest struct {
	Payload     Payload     `json:"payload"`
	Target      Target      `json:"target"`
	PushOptions PushOptions `json:"pushOptions"`
}

type DeleteRequest struct {
	NotifyId int      `json:"notifyId"`
	Token    []string `json:"token"`
}

// ==================== 响应体结构定义 ====================

// PushResponse 推送响应体
type PushResponse struct {
	Code string      `json:"code"` // 响应码，"80000000"表示成功
	Msg  string      `json:"msg"`  // 响应描述
	Data interface{} `json:"data"` // 响应数据
}
