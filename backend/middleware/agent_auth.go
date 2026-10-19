package middleware

import (
	"os"
	"strings"

	"github.com/1Panel-dev/1Panel/backend/app/repo"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/gin-gonic/gin"
)

// AgentAuth checks for Bearer token for inter-node communication
func AgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			c.Next()
			return
		}

		expectedToken := os.Getenv("PANEL_AGENT_TOKEN")
		if expectedToken == "" {
			settingRepo := repo.NewISettingRepo()
			setting, err := settingRepo.Get(settingRepo.WithByKey("AgentToken"))
			if err == nil && setting.Value != "" {
				expectedToken = setting.Value
			}
		}

		// If this node is configured with an agent token, or any matching secret token in node repo
		matched := false
		if expectedToken != "" && token == expectedToken {
			matched = true
		} else {
			// Also check if token matches any registered node secret token
			nodeRepo := repo.NewINodeRepo()
			nodes, err := nodeRepo.List()
			if err == nil {
				for _, n := range nodes {
					if n.SecretToken != "" && n.SecretToken == token {
						matched = true
						break
					}
				}
			}
		}

		if matched {
			c.Set("authMethod", constant.AuthMethodJWT)
			c.Set("isAgentRequest", true)
		}

		c.Next()
	}
}
