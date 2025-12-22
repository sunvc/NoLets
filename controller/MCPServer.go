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

// MCPServer handles MCP server requests.
// DeviceKey is passed via path parameter.
func MCPServer(c *gin.Context) {

	deviceKey := c.Param("deviceKey")

	mcpServer := server.NewMCPServer("Nolet MCP Server", common.LocalConfig.System.Version,
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	mcpServer.AddTool(mcp.NewTool("notify", getCommonToolOpts(deviceKey)...), notifyHandler)

	req := c.Request.WithContext(context.WithValue(c.Request.Context(), common.DeviceKey, deviceKey))

	server2 := server.NewStreamableHTTPServer(mcpServer)
	server2.ServeHTTP(c.Writer, req)

}

func notifyHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}

	params := &common.ParamsResult{
		Params: orderedmap.New[string, interface{}](),
		Tokens: make([]string, 0),
	}

	for k, v := range args {
		params.Params.Set(params.NormalizeKey(k), v)
	}

	common.ConvenientParamsHandler(params.Params)

	if val, ok := params.Get(common.DeviceKeys).([]string); ok {
		params.Keys = val
	}

	if val := ctx.Value(common.DeviceKey); val != nil {
		if tmpDeviceKey, ok := val.(string); ok {
			params.Keys = append(params.Keys, strings.Split(tmpDeviceKey, ",")...)
		}
	}

	for _, deviceKey := range params.Keys {
		token, err := database.DB.DeviceTokenByKey(deviceKey)
		if err == nil {
			params.Tokens = append(params.Tokens, token)
		}
	}
	params.PushType = common.ParamsNanAndDefault(params)

	if len(params.Tokens) == 0 {
		return mcp.NewToolResultError("Failed to resolve device token"), nil
	}

	if params.PushType == -1 {
		return mcp.NewToolResultError("Not Notification Body"), nil
	}

	if err := push.BatchPush(params, apns2.PushTypeAlert); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func getCommonToolOpts(deviceKey string) []mcp.ToolOption {
	tools := []mcp.ToolOption{
		mcp.WithDescription("Send a notification to a device via Nolet"),

		mcp.WithString(common.Title,
			mcp.Description("Notification title"),
		),

		mcp.WithString(common.Subtitle,
			mcp.Description("Notification subtitle"),
		),

		mcp.WithString(common.Markdown,
			mcp.Required(),
			mcp.Description("Notification body (supports Markdown)"),
		),

		mcp.WithString(common.Level,
			mcp.Description(
				"Notification level: 'critical', 'active', 'timeSensitive', or 'passive'",
			),
			mcp.Enum("critical", "active", "timeSensitive", "passive"),
		),

		mcp.WithNumber(common.Badge,
			mcp.Description("Badge number"),
		),

		mcp.WithString(common.Sound,
			mcp.Description("Notification sound"),
		),

		mcp.WithString(common.Icon,
			mcp.Description("Notification icon URL or Letters(example: B or B,ff0000) or emoji"),
		),

		mcp.WithString(common.Image,
			mcp.Description("Notification Image URL"),
		),

		mcp.WithString(common.Group,
			mcp.Description("Notification group"),
		),

		mcp.WithString(common.Url,
			mcp.Description("URL to open when the notification is tapped"),
		),

		mcp.WithString(common.Copy,
			mcp.Description("Text to copy when the copy action is triggered"),
		),
	}

	if deviceKey == "" {
		tools = append(tools,
			mcp.WithArray("deviceKeys",
				mcp.Items(
					mcp.WithString("deviceKey"),
				),
				mcp.Description("Device keys"),
			),
		)
	}

	return tools
}
