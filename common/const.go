package common

type ParamName string

const (
	CATEGORY        ParamName = "category"      // push view type  // markdown
	AUTOCOPYDEFAULT ParamName = "0"             // default auto copy
	LEVELDEFAULT    ParamName = "active"        // default push level
	DEVICEKEY       ParamName = "devicekey"     // device key
	DEVICEKEYS      ParamName = "devicekeys"    // device key list
	DEVICETOKEN     ParamName = "devicetoken"   // device token
	ID              ParamName = "id"            // ID
	TITLE           ParamName = "title"         // title
	HOST            ParamName = "host"          // host
	CALLBACK        ParamName = "callback"      // callback
	SUBTITLE        ParamName = "subtitle"      // subtitle
	CIPHERTEXT      ParamName = "ciphertext"    // ciphertext
	IMAGE           ParamName = "image"         // image
	ICON            ParamName = "icon"          // icon
	URL             ParamName = "url"           // url
	BODY            ParamName = "body"          // body
	CONTENT         ParamName = "content"       // content (compatible)
	TEXT            ParamName = "text"          // text (compatible)
	MESSAGE         ParamName = "message"       // message (compatible)
	DATA            ParamName = "data"          // data (compatible)
	GROUP           ParamName = "group"         // group
	SOUND           ParamName = "sound"         // sound
	AUTOCOPY        ParamName = "autocopy"      // auto copy
	COPY            ParamName = "copy"          // content to copy
	LEVEL           ParamName = "level"         // push level
	BADGE           ParamName = "badge"         // unread count
	MARKDOWN        ParamName = "markdown"      // whether is Markdown format
	MD              ParamName = "md"            // whether it is Markdown format (short)
	USERNAME        ParamName = "username"      // username
	PASSWORD        ParamName = "password"      // password
	PUSHGROUPNAME   ParamName = "pushgroupname" // push group name
	LOCATION        ParamName = "location"
	TTL             ParamName = "ttl"
	REPLY           ParamName = "reply"
	STYLE           ParamName = "style"
)

var ParamNames = map[ParamName]struct{}{
	CATEGORY:        {},
	AUTOCOPYDEFAULT: {},
	LEVELDEFAULT:    {},
	DEVICEKEY:       {},
	DEVICEKEYS:      {},
	DEVICETOKEN:     {},
	ID:              {},
	TITLE:           {},
	HOST:            {},
	CALLBACK:        {},
	SUBTITLE:        {},
	CIPHERTEXT:      {},
	IMAGE:           {},
	ICON:            {},
	URL:             {},
	BODY:            {},
	CONTENT:         {},
	TEXT:            {},
	MESSAGE:         {},
	DATA:            {},
	GROUP:           {},
	SOUND:           {},
	AUTOCOPY:        {},
	COPY:            {},
	LEVEL:           {},
	BADGE:           {},
	MARKDOWN:        {},
	MD:              {},
	USERNAME:        {},
	PASSWORD:        {},
	PUSHGROUPNAME:   {},
	LOCATION:        {},
	TTL:             {},
	REPLY:           {},
	STYLE:           {},
}

func (p ParamName) Name() string {
	return string(p)
}

func (p ParamName) Valid() bool {
	_, ok := ParamNames[p]
	return ok
}

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

var skipParamNames = map[ParamName]struct{}{
	DEVICEKEY:   {},
	DEVICEKEYS:  {},
	DEVICETOKEN: {},
	TITLE:       {},
	SUBTITLE:    {},
	BODY:        {},
	SOUND:       {},
	CATEGORY:    {},
}

func SkipParamNames(name string) bool {
	_, ok := skipParamNames[ParamName(name)]
	return ok
}

type CategoryStyleType string

const (
	Markdown               CategoryStyleType = "markdown"
	MyNotificationCategory CategoryStyleType = "myNotificationCategory"
	Reply                  CategoryStyleType = "reply"
	Alfa                   CategoryStyleType = "alfa"
	Bravo                  CategoryStyleType = "bravo"
	Charlie                CategoryStyleType = "charlie"
	Delta                  CategoryStyleType = "delta"
	Echo                   CategoryStyleType = "echo"
	Foxtrot                CategoryStyleType = "foxtrot"
	Golf                   CategoryStyleType = "golf"
	Hotel                  CategoryStyleType = "hotel"
	India                  CategoryStyleType = "india"
	Juliett                CategoryStyleType = "juliett"
	Kilo                   CategoryStyleType = "kilo"
	Lima                   CategoryStyleType = "lima"
	Mike                   CategoryStyleType = "mike"
	November               CategoryStyleType = "november"
	Oscar                  CategoryStyleType = "oscar"
	Papa                   CategoryStyleType = "papa"
	Quebec                 CategoryStyleType = "quebec"
	Romeo                  CategoryStyleType = "romeo"
	Sierra                 CategoryStyleType = "sierra"
	Tango                  CategoryStyleType = "tango"
	Uniform                CategoryStyleType = "uniform"
	Victor                 CategoryStyleType = "victor"
	Whiskey                CategoryStyleType = "whiskey"
	Xray                   CategoryStyleType = "xray"
	Yankee                 CategoryStyleType = "yankee"
	Zulu                   CategoryStyleType = "zulu"
)

var CategoryStyles = map[CategoryStyleType]struct{}{
	Markdown:               {},
	MyNotificationCategory: {},
	Reply:                  {},
	Alfa:                   {},
	Bravo:                  {},
	Charlie:                {},
	Delta:                  {},
	Echo:                   {},
	Foxtrot:                {},
	Golf:                   {},
	Hotel:                  {},
	India:                  {},
	Juliett:                {},
	Kilo:                   {},
	Lima:                   {},
	Mike:                   {},
	November:               {},
	Oscar:                  {},
	Papa:                   {},
	Quebec:                 {},
	Romeo:                  {},
	Sierra:                 {},
	Tango:                  {},
	Uniform:                {},
	Victor:                 {},
	Whiskey:                {},
	Xray:                   {},
	Yankee:                 {},
	Zulu:                   {},
}

func (c CategoryStyleType) Valid() bool {
	_, ok := CategoryStyles[c]
	return ok
}
