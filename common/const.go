package common

const (
	Category         = "category"               // push view type
	CategoryDefault  = "myNotificationCategory" // default category
	CategoryMarkdown = "markdown"               // markdown
	AutoCopyDefault  = "0"                      // default auto copy
	LevelDefault     = "active"                 // default push level
	DeviceKey        = "devicekey"              // device key
	DeviceKeys       = "devicekeys"             // device key list
	DeviceToken      = "devicetoken"            // device token
	ID               = "id"                     // ID
	Title            = "title"                  // title
	Host             = "host"                   // host
	Callback         = "callback"               // callback
	Subtitle         = "subtitle"               // subtitle
	CipherText       = "ciphertext"             // ciphertext
	Image            = "image"                  // image
	Icon             = "icon"                   // icon
	Url              = "url"                    // url
	Body             = "body"                   // body
	Content          = "content"                // content (compatible)
	Text             = "text"                   // text (compatible)
	Message          = "message"                // message (compatible)
	Data             = "data"                   // data (compatible)
	Group            = "group"                  // group
	Sound            = "sound"                  // sound
	AutoCopy         = "autocopy"               // auto copy
	Copy             = "copy"                   // content to copy
	Level            = "level"                  // push level
	Badge            = "badge"                  // unread count
	Markdown         = "markdown"               // whether is markdown format
	MD               = "md"                     // whether it is markdown format (short)
	UserName         = "username"               // username
	Password         = "password"               // password
	PushGroupName    = "pushgroupname"          // push group name
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
