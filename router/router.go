package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/controller"
)

func SetupRouter(engine *gin.Engine) {
	router := engine.Group(common.LocalConfig.System.URLPrefix)
	router.GET("/", controller.Home)
	router.POST("/", controller.Home)

	// Used internally by the App
	router.GET("/ping", controller.Ping)
	router.GET("/health", controller.Health)
	router.GET("/healthz", controller.Health)
	router.GET("/info", controller.Info)
	router.GET("/robots.txt", controller.RobotText)
	wellKnowGroup := router.Group("/.well-known")
	{
		wellKnowGroup.GET("/apple-app-site-association", controller.AppleSite)
	}

	// Register
	router.GET("/register/:deviceKey", GCMDecryptMiddleware(), controller.Restore)
	router.POST("/register", GCMDecryptMiddleware(), controller.Register)

	// Push requests
	router.POST("/push", controller.BasePush)

	// MCP Service
	router.Any("/mcp", controller.MCPServer)
	router.Any("/mcp/:deviceKey", controller.MCPServer)

	// title subtitle body
	router.GET("/:deviceKey/:params1/:params2/:params3", controller.BasePush)
	router.POST("/:deviceKey/:params1/:params2/:params3", controller.BasePush)
	// title body
	router.GET("/:deviceKey/:params1/:params2", controller.BasePush)
	router.POST("/:deviceKey/:params1/:params2", controller.BasePush)
	// body
	router.GET("/:deviceKey/:params1", controller.BasePush)
	router.POST("/:deviceKey/:params1", controller.BasePush)

	// Parameterized push
	router.GET("/:deviceKey", CheckDotParamMiddleware(), controller.BasePush)
	router.POST("/:deviceKey", controller.BasePush)

	engine.NoRoute(func(context *gin.Context) {
		context.AbortWithStatus(http.StatusNotFound)
	})

	engine.NoMethod(func(context *gin.Context) {
		context.AbortWithStatus(http.StatusMethodNotAllowed)
	})
}
