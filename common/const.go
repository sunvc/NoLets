package common

const (
	CATEGORY         = "category"               // push view type
	CATEGORYDEFAULT  = "myNotificationCategory" // default category
	CATEGORYMARKDOWN = "markdown"               // markdown
	AUTOCOPYDEFAULT  = "0"                      // default auto copy
	LEVELDEFAULT     = "active"                 // default push level
	DEVICEKEY        = "devicekey"              // device key
	DEVICEKEYS       = "devicekeys"             // device key list
	DEVICETOKEN      = "devicetoken"            // device token
	ID               = "id"                     // ID
	TITLE            = "title"                  // title
	HOST             = "host"                   // host
	CALLBACK         = "callback"               // callback
	SUBTITLE         = "subtitle"               // subtitle
	CIPHERTEXT       = "ciphertext"             // ciphertext
	IMAGE            = "image"                  // image
	ICON             = "icon"                   // icon
	URL              = "url"                    // url
	BODY             = "body"                   // body
	CONTENT          = "content"                // content (compatible)
	TEXT             = "text"                   // text (compatible)
	MESSAGE          = "message"                // message (compatible)
	DATA             = "data"                   // data (compatible)
	GROUP            = "group"                  // group
	SOUND            = "sound"                  // sound
	AUTOCOPY         = "autocopy"               // auto copy
	COPY             = "copy"                   // content to copy
	LEVEL            = "level"                  // push level
	BADGE            = "badge"                  // unread count
	MARKDOWN         = "markdown"               // whether is Markdown format
	MD               = "md"                     // whether it is Markdown format (short)
	USERNAME         = "username"               // username
	PASSWORD         = "password"               // password
	PUSHGROUPNAME    = "pushgroupname"          // push group name
	LOCATION         = "location"
	TTLPARAM         = "ttl"
	REPLYPARAM       = "reply"
)

const (
	HEADERCONTENTTYPE   = "Content-Type"
	HEADERUSERAGENT     = "User-Agent"
	MIMEIMAGEJPEG       = "image/jpeg"
	MIMEIMAGEPNG        = "image/png"
	MIMEIMAGESVG        = "image/svg+xml"
	MIMEAPPLICATIONJSON = "application/json"
)

const (
	APPNAME = "NoLet"
)

var SkipKeys = map[string]struct{}{
	DEVICEKEY:   {},
	DEVICEKEYS:  {},
	DEVICETOKEN: {},
	TITLE:       {},
	SUBTITLE:    {},
	BODY:        {},
	SOUND:       {},
	CATEGORY:    {},
}
