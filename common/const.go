package common

const (
	Category         = "category"               // 推送视图类型
	CategoryDefault  = "myNotificationCategory" // 模版标志
	CategoryMarkdown = "markdown"               // markdown
	AutoCopyDefault  = "0"                      // 默认自动复制
	LevelDefault     = "active"                 // 默认推送级别
	DeviceKey        = "devicekey"              // 设备key
	DeviceKeys       = "devicekeys"             // 设备key列表
	DeviceToken      = "devicetoken"            // 设备token 	// 类别
	ID               = "id"                     // ID
	Title            = "title"                  // 标题
	Host             = "host"                   // 主机
	Callback         = "callback"               // 回调
	Subtitle         = "subtitle"               // 副标题
	CipherText       = "ciphertext"             // 密文
	Image            = "image"                  // 图片
	Body             = "body"                   // 内容
	Content          = "content"                // 内容（兼容）
	Text             = "text"                   // 内容（兼容）
	Message          = "message"                // 内容（兼容）
	Data             = "data"                   // 内容（兼容）
	Group            = "group"                  // 组
	Sound            = "sound"                  // 声音
	AutoCopy         = "autocopy"               // 自动复制
	Level            = "level"                  // 等级
	Markdown         = "markdown"               // 是否是markdown格式
	MD               = "md"                     // 是否是markdown格式（简写）
	UserName         = "username"               // 用户名
	Password         = "password"               // 密码
	PushGroupName    = "pushgroupname"          // 分组推送
)

const (
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
	MIMEImageJpeg       = "image/jpeg"
	MIMEImagePng        = "image/png"
	MIMEImageSvg        = "image/svg+xml"
	MIMEApplicationJSON = "application/json"
)

const (
	APPNAME = "NoLet"
)

var SkipKeys = map[string]struct{}{
	DeviceKey:   {},
	DeviceKeys:  {},
	DeviceToken: {},
	Title:       {},
	Subtitle:    {},
	Body:        {},
	Sound:       {},
	Category:    {},
}
