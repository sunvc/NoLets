package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/database"
	"github.com/sunvc/NoLets/push"
	"github.com/sunvc/apns2"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type mcpContextKey string

const (
	mcpDeviceKeyContextKey mcpContextKey = "deviceKey"
	mcpAdminContextKey     mcpContextKey = "admin"
)

// MCPServer handles MCP server requests.
// DEVICEKEY is passed via path parameter.
func MCPServer(c *gin.Context) {

	deviceKey := c.Param("deviceKey")

	mcpServer := server.NewMCPServer("Nolet MCP Server", common.LocalConfig.System.Version,
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	mcpServer.AddTool(mcp.NewTool("notify", getCommonToolOpts(deviceKey)...), notifyHandler)

	reqCtx := context.WithValue(c.Request.Context(), mcpDeviceKeyContextKey, deviceKey)
	reqCtx = context.WithValue(reqCtx, mcpAdminContextKey, common.Admin(c))
	req := c.Request.WithContext(reqCtx)

	server2 := server.NewStreamableHTTPServer(mcpServer)
	server2.ServeHTTP(c.Writer, req)

}

func notifyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	if args == nil {
		args = map[string]any{}
	}

	params := &common.ParamsResult{
		Params: orderedmap.New[string, any](),
		Keys:   []string{},
		Users:  make([]common.User, 0),
	}

	for k, v := range args {
		normalizedKey := params.NormalizeKey(k)
		params.Params.Set(normalizedKey, normalizeMCPArgumentValue(normalizedKey, v))
	}

	common.ConvenientParamsHandler(params.Params)

	params.Keys = resolveMCPDeviceKeys(ctx, params)
	params.Users = resolveMCPUsers(ctx, params)

	params.PushType = common.ParamsNanAndDefault(params)

	if len(params.Users) == 0 {
		return mcp.NewToolResultError("Failed to resolve device token"), nil
	}

	if params.PushType == -1 {
		return mcp.NewToolResultError("Not Notification BODY"), nil
	}

	if params.PushType == 2 {
		if errs := push.LocationPush(params); len(errs) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send location push: %v", errs)), nil
		}
		return mcp.NewToolResultStructured(map[string]any{
			"status":       "ok",
			"pushType":     "location",
			"messageID":    params.GetString(common.ID),
			"resolvedKeys": params.Keys,
			"userCount":    len(params.Users),
		}, "ok"), nil
	}

	pushType := apns2.PushTypeAlert
	if params.PushType == 0 {
		pushType = apns2.PushTypeBackground
	}

	if errs := push.BatchPush(params, pushType); len(errs) > 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", errs)), nil
	}

	return mcp.NewToolResultStructured(map[string]any{
		"status":       "ok",
		"pushType":     mcpPushTypeName(pushType),
		"messageID":    params.GetString(common.ID),
		"resolvedKeys": params.Keys,
		"userCount":    len(params.Users),
	}, "ok"), nil
}

func mcpPushTypeName(pushType apns2.EPushType) string {
	if pushType == apns2.PushTypeBackground {
		return "background"
	}
	return "alert"
}

func normalizeMCPArgumentValue(key string, value any) any {
	if key != common.AUTOCOPY {
		return value
	}

	switch v := value.(type) {
	case bool:
		if v {
			return "1"
		}
		return "0"
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true":
			return "1"
		case "false":
			return "0"
		default:
			return v
		}
	case float64:
		if v != 0 {
			return "1"
		}
		return "0"
	case int:
		if v != 0 {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprint(value)
	}
}

func resolveMCPDeviceKeys(ctx context.Context, params *common.ParamsResult) []string {
	var keys []string

	if rawKeys, ok := params.Params.Get(common.DEVICEKEYS); ok {
		keys = append(keys, collectMCPStringValues(rawKeys)...)
	}

	if rawKey, ok := params.Params.Get(common.DEVICEKEY); ok {
		keys = append(keys, collectMCPStringValues(rawKey)...)
	}

	if val := ctx.Value(mcpDeviceKeyContextKey); val != nil {
		keys = append(keys, collectMCPStringValues(val)...)
	}

	keys = common.FilterShortStrings(keys, 5, 64)
	keys = common.Unique(keys)
	if len(keys) > common.LocalConfig.System.MaxDeviceKeyArrLength {
		keys = keys[:common.LocalConfig.System.MaxDeviceKeyArrLength]
	}

	return keys
}

func resolveMCPUsers(ctx context.Context, params *common.ParamsResult) []common.User {
	users := make([]common.User, 0)

	if rawToken, ok := params.Params.Get(common.DEVICETOKEN); ok {
		for _, token := range collectMCPStringValues(rawToken) {
			if len(token) > 10 {
				users = append(users, common.User{Token: token})
			}
		}
	}

	for _, deviceKey := range params.Keys {
		user, err := database.DB.DeviceTokenByKey(deviceKey)
		if err == nil {
			users = append(users, *user)
		}
	}

	if val := ctx.Value(mcpAdminContextKey); val != nil {
		if isAdmin, ok := val.(bool); ok && isAdmin {
			if rawGroup, ok := params.Params.Get(common.PUSHGROUPNAME); ok {
				for _, group := range collectMCPStringValues(rawGroup) {
					groupUsers, err := database.DB.DeviceTokenByGroup(group)
					if err != nil {
						continue
					}
					for _, user := range groupUsers {
						users = append(users, *user)
					}
				}
			}
		}
	}

	return common.UserUnique(users)
}

func collectMCPStringValues(raw any) []string {
	switch v := raw.(type) {
	case string:
		return splitAndTrimCSV(v)
	case []string:
		values := make([]string, 0, len(v))
		for _, item := range v {
			values = append(values, splitAndTrimCSV(item)...)
		}
		return values
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			values = append(values, collectMCPStringValues(item)...)
		}
		return values
	default:
		return splitAndTrimCSV(fmt.Sprint(raw))
	}
}

func splitAndTrimCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func getCommonToolOpts(deviceKey string) []mcp.ToolOption {
	tools := []mcp.ToolOption{
		mcp.WithDescription("Send notifications to iOS devices through NoLets. Supports alert, background, and location pushes."),

		mcp.WithString(common.TITLE,
			mcp.Description("Notification title"),
		),

		mcp.WithString(common.SUBTITLE,
			mcp.Description("Notification subtitle"),
		),

		mcp.WithString(common.BODY,
			mcp.Description("Notification body text"),
		),

		mcp.WithString(common.CONTENT,
			mcp.Description("Alias of body"),
		),

		mcp.WithString(common.TEXT,
			mcp.Description("Alias of body"),
		),

		mcp.WithString(common.MESSAGE,
			mcp.Description("Alias of body"),
		),

		mcp.WithString(common.DATA,
			mcp.Description("Alias of body"),
		),

		mcp.WithString(common.MARKDOWN,
			mcp.Description("Markdown body, automatically sets category to markdown"),
		),

		mcp.WithString(common.MD,
			mcp.Description("Short alias of markdown"),
		),

		mcp.WithString(common.CATEGORY,
			mcp.Description("Notification category"),
			mcp.Enum(common.CATEGORYDEFAULT, common.CATEGORYMARKDOWN),
		),

		mcp.WithString(common.CIPHERTEXT,
			mcp.Description("Encrypted content payload"),
		),

		mcp.WithString(common.LEVEL,
			mcp.Description(
				"Notification level: 'critical', 'active', 'timeSensitive', or 'passive'",
			),
			mcp.Enum("critical", "active", "timeSensitive", "passive"),
		),

		mcp.WithNumber(common.BADGE,
			mcp.Description("Badge number"),
		),

		mcp.WithString(common.SOUND,
			mcp.Description("Notification sound"),
		),

		mcp.WithString(common.ICON,
			mcp.Description("Notification icon URL or Letters(example: B or B,ff0000) or emoji"),
		),

		mcp.WithString(common.IMAGE,
			mcp.Description("Notification image URL"),
		),

		mcp.WithString(common.GROUP,
			mcp.Description("Notification group"),
		),

		mcp.WithString(common.URL,
			mcp.Description("URL to open when the notification is tapped"),
		),

		mcp.WithString(common.COPY,
			mcp.Description("Text to copy when the copy action is triggered"),
		),

		mcp.WithString(common.AUTOCOPY,
			mcp.Description("Automatically copy content: '1' to enable, '0' to disable"),
			mcp.Enum("0", "1"),
		),

		mcp.WithString(common.ID,
			mcp.Description("Message ID used for collapse and deduplication"),
		),

		mcp.WithString(common.LOCATION,
			mcp.Description("Location query URL; when provided, sends a location push"),
		),

		mcp.WithString(common.DEVICETOKEN,
			mcp.Description("Direct APNs device token"),
		),

		mcp.WithString(common.PUSHGROUPNAME,
			mcp.Description("Push to all devices in the given group (admin only)"),
		),

		mcp.WithString(common.CALLBACK,
			mcp.Description("Custom callback value forwarded to the client"),
		),

		mcp.WithNumber(common.TTLPARAM,
			mcp.Description("Time to live for the notification payload"),
		),

		mcp.WithString(common.REPLYPARAM,
			mcp.Description("Reply text or reply payload forwarded to the client"),
		),
	}

	if deviceKey == "" {
		tools = append(tools,
			mcp.WithString(common.DEVICEKEY,
				mcp.Description("Single device key"),
			),
			mcp.WithArray(common.DEVICEKEYS,
				mcp.WithStringItems(),
				mcp.Description("Device keys"),
			),
		)
	} else {
		tools = append(tools,
			mcp.WithString(common.DEVICEKEY,
				mcp.Description("Optional extra device key"),
			),
			mcp.WithArray(common.DEVICEKEYS,
				mcp.WithStringItems(),
				mcp.Description("Optional extra device keys"),
			),
		)
	}

	return tools
}
