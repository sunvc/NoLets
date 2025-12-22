package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// ParamsResult struct is used to store and manage request parameters.
// It uses an ordered map to store parameters, ensuring the processing order of parameters.
type ParamsResult struct {
	Params   *orderedmap.OrderedMap[string, interface{}]
	Tokens   []string
	Keys     []string
	PushType int
}

// Get parameter value
// params:
//   - key: parameter key
//
// return:
//   - interface{}: parameter value, returns empty string if not exists
func (p *ParamsResult) Get(key string) interface{} {
	if value, ok := p.Params.Get(key); ok {
		return value
	}
	return ""
}

func (p *ParamsResult) GetString(key string) string {
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
func (p *ParamsResult) NormalizeKey(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9',
			c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z':
			b = append(b, c)
		}
	}
	return strings.ToLower(string(b))
}

// NewParamsResult creates a new parameter result object.
// Parameters:
//   - c: The gin context object, used to get request parameters.
//
// Returns:
//   - *ParamsResult: The initialized parameter result object.
func NewParamsResult(c *gin.Context) *ParamsResult {
	main := &ParamsResult{
		Params: orderedmap.New[string, interface{}](),
		Keys:   []string{},
		Tokens: []string{},
	}
	main.HandlerParamsToMapOrder(c)
	main.PushType = ParamsNanAndDefault(main)

	var resultKeys []string

	if keys, ok := main.Params.Get(DeviceKeys); ok {
		if vals, oka := keys.([]interface{}); oka {
			resultKeys = InterfaceSliceToStringSlice(vals)
		}
	}

	if key, ok := main.Params.Get(DeviceKey); ok {
		if val, oka := key.(string); oka {
			resultKeys = append(resultKeys, strings.Split(val, ",")...)
		}
	}

	resultKeys = FilterShortStrings(resultKeys, 5, 64)
	main.Keys = Unique[string](resultKeys)

	if len(main.Keys) > LocalConfig.System.MaxDeviceKeyArrLength {
		main.Keys = main.Keys[:LocalConfig.System.MaxDeviceKeyArrLength]
	}

	var tokens []string
	if token, ok := main.Params.Get(DeviceToken); ok {
		if val, oka := token.(string); oka && len(val) > 10 {
			tokens = append(tokens, val)
		}
	}

	tokens = FilterShortStrings(tokens, 60, 65)

	main.Tokens = tokens

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
	result := orderedmap.New[string, interface{}]()

	// Check if it is an admin
	host := GetClientHost(c)
	if Admin(c) {
		result.Set(Host, host)
	}
	// Compatible with old versions
	result.Set(Callback, host)

	getDeviceKey := func(value string) {
		deviceKeys := strings.Split(value, ",")
		if len(deviceKeys) > 1 {
			result.Set(DeviceKeys, deviceKeys)
		} else {
			result.Set(DeviceKey, value)
		}
	}

	switch len(c.Params) {
	case 1:
		getDeviceKey(c.Params[0].Value)
	case 2:
		getDeviceKey(c.Params[0].Value)
		result.Set(Body, c.Params[1].Value)
	case 3:
		getDeviceKey(c.Params[0].Value)
		result.Set(Title, c.Params[1].Value)
		result.Set(Body, c.Params[2].Value)
	case 4:
		getDeviceKey(c.Params[0].Value)
		result.Set(Title, c.Params[1].Value)
		result.Set(Subtitle, c.Params[2].Value)
		result.Set(Body, c.Params[3].Value)
	}

	// parse query args (medium priority)
	{
		var keys []string
		var params = c.Request.URL.Query()
		for key, values := range params {
			lowKey := p.NormalizeKey(key)
			if len(values) > 0 {
				if lowKey == DeviceKey {
					keys = append(keys, values...)
				} else {
					result.Set(lowKey, values[0])
				}
			}

		}

		if keysNum := len(keys); keysNum > 0 {
			if keysNum == 1 {
				result.Set(DeviceKey, keys[0])
			} else {
				result.Set(DeviceKeys, keys)
			}
		}
	}

	// POST Body
	if c.Request.Method == http.MethodPost {

		contentType := c.Request.Header.Get(HeaderContentType)
		if strings.HasPrefix(contentType, MIMEApplicationJSON) {
			var jsonData map[string]interface{}
			err := c.ShouldBindBodyWithJSON(&jsonData)
			if err == nil {
				for k, v := range jsonData {
					result.Set(p.NormalizeKey(k), v)
				}
			}
		} else {
			err := c.Request.ParseForm()
			if err == nil {
				for k, v := range c.Request.PostForm {
					result.Set(p.NormalizeKey(k), v)
				}
			}
		}
	}

	ConvenientParamsHandler(result)

	// Write to ParamsResult.Params
	for pair := result.Oldest(); pair != nil; pair = pair.Next() {
		p.Params.Set(p.NormalizeKey(pair.Key), pair.Value)
	}
}

func ConvenientParamsHandler(result *orderedmap.OrderedMap[string, interface{}]) {
	// Try to convert from other fields first
	if data, dataOk := result.Get(Data); dataOk {
		result.Set(Body, fmt.Sprint(data))
		result.Delete(Data)
	} else if content, contentOk := result.Get(Content); contentOk {
		result.Set(Body, fmt.Sprint(content))
		result.Delete(Content)
	} else if message, messageOk := result.Get(Message); messageOk {
		result.Set(Body, fmt.Sprint(message))
		result.Delete(Message)
	} else if text, textOk := result.Get(Text); textOk {
		result.Set(Body, fmt.Sprint(text))
		result.Delete(Text)
	}

	// Process markdown fields
	// If markdown field exists, convert it to body and set category to markdown
	if v, ok := result.Get(Markdown); ok {
		result.Set(Body, fmt.Sprint(v))
		result.Set(Category, CategoryMarkdown)
		result.Delete(Markdown)

	}
	// If md field exists, convert it to body and set category to markdown
	if v, ok := result.Get(MD); ok {
		result.Set(Body, fmt.Sprint(v))
		result.Set(Category, CategoryMarkdown)
		result.Delete(MD)
	}

	// Normalize category field
	// If category is not default or markdown, set it to default
	if v, ok := result.Get(Category); ok {
		if v != CategoryDefault && v != CategoryMarkdown {
			result.Set(Category, CategoryDefault)
		}
	}

	// Process sound file suffix
	// If the sound file does not have a .caf suffix, add it
	if val, ok := result.Get(Sound); ok {
		if sound, oka := val.(string); oka {
			if !strings.HasSuffix(sound, ".caf") {
				result.Set(Sound, fmt.Sprintf("%v.caf", sound))
			}
		}
	}
}

func ParamsNanAndDefault(paramsResult *ParamsResult) (resultType int) {
	get := func(key string) bool {
		v, ok := paramsResult.Params.Get(key)
		if !ok || v == nil {
			return true
		}
		return len(strings.TrimSpace(fmt.Sprint(v))) == 0
	}

	titleNan := get(Title)
	subTitleNan := get(Subtitle)
	bodyNan := get(Body)
	cipherNan := get(CipherText)
	imageNan := get(Image)
	idNan := get(ID)

	contentNan := titleNan && subTitleNan && bodyNan && cipherNan && imageNan

	// ---- resultType logic ----
	switch {
	case contentNan && !idNan:
		resultType = 0
	case !contentNan:
		resultType = 1
	default:
		resultType = -1
		return
	}

	// ---- Logic to supplement body: "-" ----
	if (!cipherNan || !imageNan) && titleNan && subTitleNan && bodyNan {
		paramsResult.Params.Set(Body, "-")
	}

	// ---- Default value processing ----
	setDefault := func(key string, defaultValue interface{}) {
		realKey := paramsResult.NormalizeKey(key)
		if v, ok := paramsResult.Params.Get(realKey); !ok || v == nil || len(strings.TrimSpace(fmt.Sprint(v))) == 0 {
			paramsResult.Params.Set(realKey, defaultValue)
		}
	}

	setDefault(AutoCopy, AutoCopyDefault)
	setDefault(Level, LevelDefault)
	setDefault(Category, CategoryDefault)
	setDefault(ID, func() interface{} {
		messageID, _ := uuid.NewUUID()
		return messageID.String()
	}())

	return
}
