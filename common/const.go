package common

type ParamName = string

const (
	CATEGORY         ParamName = "category"               // push view type
	CATEGORYDEFAULT  ParamName = "myNotificationCategory" // default category
	CATEGORYMARKDOWN ParamName = "markdown"               // markdown
	AUTOCOPYDEFAULT  ParamName = "0"                      // default auto copy
	LEVELDEFAULT     ParamName = "active"                 // default push level
	DEVICEKEY        ParamName = "devicekey"              // device key
	DEVICEKEYS       ParamName = "devicekeys"             // device key list
	DEVICETOKEN      ParamName = "devicetoken"            // device token
	ID               ParamName = "id"                     // ID
	TITLE            ParamName = "title"                  // title
	HOST             ParamName = "host"                   // host
	CALLBACK         ParamName = "callback"               // callback
	SUBTITLE         ParamName = "subtitle"               // subtitle
	CIPHERTEXT       ParamName = "ciphertext"             // ciphertext
	IMAGE            ParamName = "image"                  // image
	ICON             ParamName = "icon"                   // icon
	URL              ParamName = "url"                    // url
	BODY             ParamName = "body"                   // body
	CONTENT          ParamName = "content"                // content (compatible)
	TEXT             ParamName = "text"                   // text (compatible)
	MESSAGE          ParamName = "message"                // message (compatible)
	DATA             ParamName = "data"                   // data (compatible)
	GROUP            ParamName = "group"                  // group
	SOUND            ParamName = "sound"                  // sound
	AUTOCOPY         ParamName = "autocopy"               // auto copy
	COPY             ParamName = "copy"                   // content to copy
	LEVEL            ParamName = "level"                  // push level
	BADGE            ParamName = "badge"                  // unread count
	MARKDOWN         ParamName = "markdown"               // whether is Markdown format
	MD               ParamName = "md"                     // whether it is Markdown format (short)
	USERNAME         ParamName = "username"               // username
	PASSWORD         ParamName = "password"               // password
	PUSHGROUPNAME    ParamName = "pushgroupname"          // push group name
	LOCATION         ParamName = "location"
	TTL              ParamName = "ttl"
	REPLY            ParamName = "reply"
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
