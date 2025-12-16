package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wk8/go-ordered-map/v2"
)

// ParamsResult 结构体用于存储和管理请求参数
// 使用有序映射存储参数，保证参数的处理顺序
type ParamsResult struct {
	Params   *orderedmap.OrderedMap[string, interface{}]
	Tokens   []string
	Keys     []string
	PushType int
}

// Get 获取参数值
// 参数:
//   - key: 参数键名
//
// 返回:
//   - interface{}: 参数值，如果不存在则返回空字符串
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

// NormalizeKey 规范化参数键名
// 主要功能:
// 1. 去除所有的符号,空格
// 2. 转为小写, 只保留数字,字母
// 参数:
//   - s: 需要规范化的键名字符串
//
// 返回:
//   - string: 规范化后的键名
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

// NewParamsResult 创建新的参数结果对象
// 参数:
//   - c: gin上下文对象，用于获取请求参数
//
// 返回:
//   - *ParamsResult: 初始化后的参数结果对象
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
			resultKeys = append(resultKeys, val)
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

// HandlerParamsToMapOrder 处理请求参数并转换为有序映射
// 主要功能：
// 1. 从URL路径参数中提取设备密钥、标题、副标题和内容
// 2. 从URL查询参数中获取额外参数
// 3. 处理POST请求的表单数据和JSON数据
// 4. 对参数进行便捷处理
// 5. 将处理后的参数保存到有序映射中
func (p *ParamsResult) HandlerParamsToMapOrder(c *gin.Context) {
	result := orderedmap.New[string, interface{}]()

	// 判断是否是管理员
	host := GetClientHost(c)
	if Admin(c) {
		result.Set(Host, host)
	}
	// 兼容旧版本
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

	// 先尝试从其他字段转换
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

	// 处理 markdown 字段
	// 如果存在 markdown 字段，将其转换为 body 并设置 category 为 markdown
	if v, ok := result.Get(Markdown); ok {
		result.Set(Body, fmt.Sprint(v))
		result.Set(Category, CategoryMarkdown)
		result.Delete(Markdown)

	}
	// 如果存在 md 字段，将其转换为 body 并设置 category 为 markdown
	if v, ok := result.Get(MD); ok {
		result.Set(Body, fmt.Sprint(v))
		result.Set(Category, CategoryMarkdown)
		result.Delete(MD)
	}

	// 规范化 category 字段
	// 如果 category 不是默认值或 markdown，则设置为默认值
	if v, ok := result.Get(Category); ok {
		if v != CategoryDefault && v != CategoryMarkdown {
			result.Set(Category, CategoryDefault)
		}
	}

	// 处理声音文件后缀
	// 如果声音文件没有 .caf 后缀，则添加后缀
	if val, ok := result.Get(Sound); ok {
		if sound, oka := val.(string); oka {
			if !strings.HasSuffix(sound, ".caf") {
				result.Set(Sound, fmt.Sprintf("%v.caf", sound))
			}
		}
	}

	// 写入 ParamsResult.Params
	for pair := result.Oldest(); pair != nil; pair = pair.Next() {
		p.Params.Set(p.NormalizeKey(pair.Key), pair.Value)
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

	// ---- resultType 逻辑 ----
	switch {
	case contentNan && !idNan:
		resultType = 0
	case !contentNan:
		resultType = 1
	default:
		resultType = -1
		return
	}

	// ---- 补充 body: "-" 的逻辑 ----
	if (!cipherNan || !imageNan) && titleNan && subTitleNan && bodyNan {
		paramsResult.Params.Set(Body, "-")
	}

	// ---- 默认值处理 ----
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
