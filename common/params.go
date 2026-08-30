package common

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sunvc/apns2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// ParamsResult struct is used to store and manage request parameters.
// It uses an ordered map to store parameters, ensuring the processing order of parameters.
type ParamsResult struct {
	Params   *orderedmap.OrderedMap[ParamName, interface{}]
	Users    []User
	Keys     []string
	PushType apns2.EPushType
}

// Get parameter value
// params:
//   - key: parameter key
//
// return:
//   - interface{}: parameter value, returns empty string if not exists
func (p *ParamsResult) Get(key ParamName) interface{} {
	if value, ok := p.Params.Get(key); ok {
		return value
	}
	return ""
}

func (p *ParamsResult) GetString(key ParamName) string {
	if value, ok := p.Params.Get(key); ok {
		return fmt.Sprint(value)
	}
	return ""
}

// NormalizeKey normalizes the parameter key.
// Main functions:
// 1. Removes all symbols and spaces.
// 2. Converts to lowercase, keeping only numbers and letters.
// Parameters:
//   - s: The key string to be normalized.
//
// Returns:
//   - string: The normalized key.
func (p *ParamsResult) NormalizeKey(s string) ParamName {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+('a'-'A'))
		case c >= '0' && c <= '9', c >= 'a' && c <= 'z':
			b = append(b, c)
		}
	}
	return ParamName(b)
}

// NewParamsResult creates a new parameter result object.
// Parameters:
//   - c: The gin context object, used to get request parameters.
//
// Returns:
//   - *ParamsResult: The initialized parameter result object.
func NewParamsResult(c *gin.Context) *ParamsResult {
	main := &ParamsResult{
		Params: orderedmap.New[ParamName, interface{}](),
		Keys:   []string{},
		Users:  make([]User, 0),
	}
	main.HandlerParamsToMapOrder(c)
	main.PushType = ParamsNanAndDefault(main)

	main.Keys = FilterShortStrings(main.Keys, 5, 64)

	var users []User
	if token, ok := main.Params.Get(DEVICETOKEN); ok {
		if val, oka := token.(string); oka && len(val) > 10 {
			users = append(users, User{Token: val})
		}
	}

	users = UserUnique(users)
	main.Users = users
	return main
}

// HandlerParamsToMapOrder processes request parameters and converts them to an ordered map.
// Main functions:
// 1. Extracts device key, title, subtitle, and content from URL path parameters.
// 2. Gets additional parameters from URL query parameters.
// 3. Processes form data and JSON data from POST requests.
// 4. Performs convenient processing on parameters.
// 5. Saves the processed parameters into an ordered map.
func (p *ParamsResult) HandlerParamsToMapOrder(c *gin.Context) {
	result := p.Params

	// Check if it is an admin
	host := GetClientHost(c)
	if Admin(c) {
		result.Set(HOST, host)
	}

	getDeviceKey := func(value string) {
		p.Keys = append(p.Keys, strings.Split(value, ",")...)
	}

	switch len(c.Params) {
	case 1:
		getDeviceKey(c.Params[0].Value)
	case 2:
		getDeviceKey(c.Params[0].Value)
		result.Set(BODY, c.Params[1].Value)
	case 3:
		getDeviceKey(c.Params[0].Value)
		result.Set(TITLE, c.Params[1].Value)
		result.Set(BODY, c.Params[2].Value)
	case 4:
		getDeviceKey(c.Params[0].Value)
		result.Set(TITLE, c.Params[1].Value)
		result.Set(SUBTITLE, c.Params[2].Value)
		result.Set(BODY, c.Params[3].Value)
	}

	// device keys can arrive via query string or request body
	var keys []string

	// parse query args (medium priority)
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 {
			continue
		}
		lowKey := p.NormalizeKey(key)
		if lowKey == DEVICEKEY || lowKey == DEVICEKEYS {
			keys = append(keys, values...)
		} else {
			result.Set(lowKey, values[0])
		}
	}

	// POST BODY
	if c.Request.Method == http.MethodPost {

		contentType := c.Request.Header.Get(HEADERCONTENTTYPE)
		if strings.HasPrefix(contentType, MIMEAPPLICATIONJSON) {
			var jsonData map[string]interface{}
			err := c.ShouldBindBodyWithJSON(&jsonData)
			if err == nil {
				for k, v := range jsonData {
					lowKey := p.NormalizeKey(k)
					if lowKey == DEVICEKEY || lowKey == DEVICEKEYS {
						keys = append(keys, bodyDeviceKeys(v)...)
					} else {
						result.Set(lowKey, v)
					}
				}
			}
		} else {
			err := c.Request.ParseForm()
			if err == nil {
				for k, values := range c.Request.PostForm {
					lowKey := p.NormalizeKey(k)
					if lowKey == DEVICEKEY || lowKey == DEVICEKEYS {
						keys = append(keys, values...)
					} else if len(values) > 0 {
						result.Set(lowKey, values[0])
					}
				}
			}
		}
	}

	p.Keys = append(p.Keys, keys...)

	ConvenientParamsHandler(result)
}

// bodyDeviceKeys extracts device keys from a JSON body value:
// a comma-separated string, a string array, or any other value stringified.
func bodyDeviceKeys(v interface{}) []string {
	switch val := v.(type) {
	case string:
		return strings.Split(val, ",")
	case []string:
		return val
	case []interface{}:
		keys := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				keys = append(keys, s)
			}
		}
		return keys
	default:
		return nil
	}
}

func ConvenientParamsHandler(result *orderedmap.OrderedMap[ParamName, interface{}]) {
	// Body aliases: the first present one wins
	for _, key := range []ParamName{DATA, CONTENT, MESSAGE, TEXT} {
		if v, ok := result.Get(key); ok {
			result.Set(BODY, fmt.Sprint(v))
			result.Delete(key)
			break
		}
	}

	// markdown/md: use content as body and mark the style as markdown
	for _, key := range []ParamName{MARKDOWN, MD} {
		if v, ok := result.Get(key); ok {
			result.Set(BODY, fmt.Sprint(v))
			result.Set(STYLE, Markdown)
			result.Delete(key)
		}
	}

	// Sound file: append .caf suffix if missing
	if sound, ok := result.Get(SOUND); ok {
		if s, ok := sound.(string); ok && !strings.HasSuffix(s, ".caf") {
			result.Set(SOUND, s+".caf")
		}
	}
}

func ParamsNanAndDefault(paramsResult *ParamsResult) (resultType apns2.EPushType) {
	get := func(key ParamName) bool {
		v, ok := paramsResult.Params.Get(key)
		return !ok || v == nil || strings.TrimSpace(fmt.Sprint(v)) == ""
	}

	titleNan := get(TITLE)
	subTitleNan := get(SUBTITLE)
	bodyNan := get(BODY)
	cipherNan := get(CIPHERTEXT)
	imageNan := get(IMAGE)
	idNan := get(ID)

	location := func() bool {
		v, ok := paramsResult.Params.Get(LOCATION)
		if !ok {
			return false
		}

		u, err := url.Parse(fmt.Sprint(v))
		if err != nil {
			return false
		}

		return u.Scheme != "" && u.Host != ""
	}()

	contentNan := titleNan && subTitleNan && bodyNan && cipherNan && imageNan

	// ---- resultType logic ----
	switch {
	case location:
		resultType = apns2.PushTypeLocation
	case contentNan && !idNan:
		resultType = apns2.PushTypeBackground
	case !contentNan:
		resultType = apns2.PushTypeAlert
	default:
		resultType = "0"
		return
	}

	// ---- Logic to supplement body: "-" ----
	if (!cipherNan || !imageNan) && titleNan && subTitleNan && bodyNan {
		paramsResult.Params.Set(BODY, "-")
	}

	// ---- Default value processing ----
	setDefault := func(key ParamName, defaultValue interface{}) {
		if v, ok := paramsResult.Params.Get(key); !ok || v == nil || strings.TrimSpace(fmt.Sprint(v)) == "" {
			paramsResult.Params.Set(key, defaultValue)
		}
	}

	v, ok := paramsResult.Params.Get(CATEGORY)
	if !ok || !CategoryStyleType(fmt.Sprint(v)).Valid() {
		paramsResult.Params.Set(CATEGORY, MyNotificationCategory)
	}

	setDefault(AUTOCOPY, AUTOCOPYDEFAULT)
	setDefault(LEVEL, LEVELDEFAULT)
	setDefault(ID, uuid.NewString())

	return
}
