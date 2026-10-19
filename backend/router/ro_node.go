package router

import (
	v1 "github.com/1Panel-dev/1Panel/backend/app/api/v1"
	"github.com/1Panel-dev/1Panel/backend/middleware"
	"github.com/gin-gonic/gin"
)

type NodeRouter struct{}

func (s *NodeRouter) InitRouter(Router *gin.RouterGroup) {
	nodeRouter := Router.Group("nodes").
		Use(middleware.JwtAuth()).
		Use(middleware.SessionAuth()).
		Use(middleware.PasswordExpired())
	baseApi := v1.ApiGroupApp.BaseApi
	{
		nodeRouter.GET("", baseApi.ListNode)
		nodeRouter.GET("/token", baseApi.GetLocalAgentToken)
		nodeRouter.POST("/search", baseApi.SearchNode)
		nodeRouter.POST("", baseApi.CreateNode)
		nodeRouter.POST("/update", baseApi.UpdateNode)
		nodeRouter.POST("/del", baseApi.DeleteNode)
		nodeRouter.POST("/test", baseApi.TestNode)
	}
}
