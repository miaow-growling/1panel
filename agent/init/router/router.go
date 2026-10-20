package router

import (
	"os"
	"strings"

	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"github.com/1Panel-dev/1Panel/agent/middleware"
	rou "github.com/1Panel-dev/1Panel/agent/router"
	"github.com/gin-gonic/gin"
)

var (
	Router *gin.Engine
)

func Routers() *gin.Engine {
	Router = gin.Default()
	Router.Use(i18n.UseI18n())

	PrivateGroup := Router.Group("/api/v2")
	if !global.IsMaster {
		PrivateGroup.Use(middleware.Certificate())
	}
	PrivateGroup.Use(middleware.OperationResolveMeta())
	for _, router := range rou.RouterGroupApp {
		router.InitRouter(PrivateGroup)
	}
	PrivateGroup.GET("/health/check", v2.ApiGroupApp.BaseApi.CheckHealth)

	// Public liveness probe for node mode; a presented Bearer token must be
	// valid, which is how the master verifies the secret at registration.
	Router.GET("/health", func(c *gin.Context) {
		if global.IsMaster {
			c.JSON(200, gin.H{"status": "ok"})
			return
		}
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(200, gin.H{"status": "ok"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		expected := os.Getenv("PANEL_AGENT_TOKEN")
		if expected == "" {
			if value, err := repo.NewISettingRepo().GetValueByKey("AgentToken"); err == nil {
				expected = value
			}
		}
		if expected == "" || token != expected {
			c.JSON(401, gin.H{"code": 401, "message": "secret token mismatch"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	return Router
}
