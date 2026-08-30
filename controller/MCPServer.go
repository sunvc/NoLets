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

	mcpServer.AddTool(mcp.NewTool("notify", getCommonToolOpts()...), notifyHandler)

	reqCtx := context.WithValue(c.Request.Context(), mcpDeviceKeyContextKey, deviceKey)
	reqCtx = context.WithValue(reqCtx, mcpAdminContextKey, common.Admin(c))
	req := c.Request.WithContext(reqCtx)

	server2 := server.NewStreamableHTTPServer(mcpServer)
	server2.ServeHTTP(c.Writer, req)

}

func notifyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := &common.ParamsResult{
		Params: orderedmap.New[common.ParamName, any](),
		Keys:   []string{},
		Users:  make([]common.User, 0),
	}

	for k, v := range request.GetArguments() {
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

	if params.PushType == "0" {
		return mcp.NewToolResultError("Not Notification BODY"), nil
	}

	var pushTypeName string
	if params.PushType == apns2.PushTypeLocation {
		if errs := push.LocationPush(params); len(errs) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send location push: %v", errs)), nil
		}
		pushTypeName = "location"
	} else {
		if errs := push.BatchPush(params, params.PushType); len(errs) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", errs)), nil
		}
		pushTypeName = mcpPushTypeName(params.PushType)
	}

	return mcp.NewToolResultStructured(map[string]any{
		"status":       "ok",
		"pushType":     pushTypeName,
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

func normalizeMCPArgumentValue(key common.ParamName, value any) any {
	if key != common.AUTOCOPY {
		return value
	}

	oneZero := func(on bool) string {
		if on {
			return "1"
		}
		return "0"
	}

	switch v := value.(type) {
	case bool:
		return oneZero(v)
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true":
			return "1"
		case "false":
			return "0"
		default:
			return v
		}
	case float64: // JSON numbers
		return oneZero(v != 0)
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

func getCommonToolOpts() []mcp.ToolOption {
	str := func(name common.ParamName, desc string, enum ...string) mcp.ToolOption {
		opts := []mcp.PropertyOption{mcp.Description(desc)}
		if len(enum) > 0 {
			opts = append(opts, mcp.Enum(enum...))
		}
		return mcp.WithString(string(name), opts...)
	}
	num := func(name common.ParamName, desc string) mcp.ToolOption {
		return mcp.WithNumber(string(name), mcp.Description(desc))
	}
	arr := func(name common.ParamName, desc string) mcp.ToolOption {
		return mcp.WithArray(string(name), mcp.WithStringItems(), mcp.Description(desc))
	}

	return []mcp.ToolOption{
		mcp.WithDescription("Send notifications to iOS devices through NoLets. Supports alert, background, and location pushes."),

		str(common.TITLE, "Notification title"),
		str(common.SUBTITLE, "Notification subtitle"),
		str(common.BODY, "Notification body text"),
		str(common.CONTENT, "Alias of body"),
		str(common.TEXT, "Alias of body"),
		str(common.MESSAGE, "Alias of body"),
		str(common.DATA, "Alias of body"),
		str(common.MARKDOWN, "Markdown body, automatically sets category to markdown"),
		str(common.MD, "Short alias of markdown"),
		str(common.CATEGORY, "Notification category",
			string(common.MyNotificationCategory), string(common.Markdown)),
		str(common.CIPHERTEXT, "Encrypted content payload"),
		str(common.LEVEL, "Notification level: 'critical', 'active', 'timeSensitive', or 'passive'",
			"critical", "active", "timeSensitive", "passive"),
		num(common.BADGE, "Badge number"),
		str(common.SOUND, "Notification sound"),
		str(common.ICON, "Notification icon URL or Letters(example: B or B,ff0000) or emoji"),
		str(common.IMAGE, "Notification image URL"),
		str(common.GROUP, "Notification group"),
		str(common.URL, "URL to open when the notification is tapped"),
		str(common.COPY, "Text to copy when the copy action is triggered"),
		str(common.AUTOCOPY, "Automatically copy content: '1' to enable, '0' to disable", "0", "1"),
		str(common.ID, "Message ID used for collapse and deduplication"),
		str(common.LOCATION, "Location query URL; when provided, sends a location push"),
		str(common.DEVICETOKEN, "Direct APNs device token"),
		str(common.PUSHGROUPNAME, "Push to all devices in the given group (admin only)"),
		str(common.CALLBACK, "Custom callback value forwarded to the client"),
		num(common.TTL, "Time to live for the notification payload"),
		str(common.REPLY, "Reply text or reply payload forwarded to the client"),
		str(common.DEVICEKEY, "Device key (optional when already passed in the URL path)"),
		arr(common.DEVICEKEYS, "Device keys"),
	}
}
