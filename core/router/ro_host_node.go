package router

import (
	v2 "github.com/1Panel-dev/1Panel/core/app/api/v2"
	"github.com/1Panel-dev/1Panel/core/middleware"
	"github.com/gin-gonic/gin"
)

type HostNodeRouter struct{}

func (a *HostNodeRouter) InitRouter(Router *gin.RouterGroup) {
	nodeRouter := Router.Group("nodes").
		Use(middleware.SessionAuth()).
		Use(middleware.PasswordExpired())

	baseApi := v2.ApiGroupApp.BaseApi
	{
		nodeRouter.GET("", baseApi.ListHostNodes)
		nodeRouter.POST("/list", baseApi.ListHostNodesForFrontend)
		nodeRouter.GET("/simple/all", baseApi.GetHostNodesSimple)
		nodeRouter.POST("", baseApi.CreateHostNode)
		nodeRouter.POST("/update", baseApi.UpdateHostNode)
		nodeRouter.POST("/del", baseApi.DeleteHostNode)
		nodeRouter.POST("/test", baseApi.TestHostNode)
		nodeRouter.POST("/test/id", baseApi.TestHostNodeByID)
		nodeRouter.POST("/install-script", baseApi.GetHostNodeInstallScript)

		// One-time-token download outside SessionAuth; PasswordExpired exempts
		// this path so a fresh node can fetch the binary pre-registration.
		binaryRouter := Router.Group("nodes")
		{
			binaryRouter.GET("/agent-binary", baseApi.DownloadAgentBinary)
		}
	}
}
