package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/backend/app/repo"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/gin-gonic/gin"
)

// MaxAgentClockSkew limits how old (or far in the future) a signed agent
// request may be, mirroring the official v2 replay protection design.
const MaxAgentClockSkew = 5 * time.Minute

// PanelForwardedUserKey carries the master-side username forwarded through
// the X-Panel-User header (same pattern as 1Panel v2 core -> agent).
const PanelForwardedUserKey = "panelForwardedUser"

func resolveAgentToken() string {
	if envToken := os.Getenv("PANEL_AGENT_TOKEN"); envToken != "" {
		return envToken
	}
	settingRepo := repo.NewISettingRepo()
	setting, err := settingRepo.Get(settingRepo.WithByKey("AgentToken"))
	if err == nil && setting.Value != "" {
		return setting.Value
	}
	return ""
}

// MatchAgentToken reports whether the presented bearer token is trusted by
// this node (its own agent token, or any registered node secret token).
func MatchAgentToken(token string) bool {
	if token == "" {
		return false
	}
	if expected := resolveAgentToken(); expected != "" && token == expected {
		return true
	}
	nodeRepo := repo.NewINodeRepo()
	nodes, err := nodeRepo.List()
	if err == nil {
		for _, n := range nodes {
			if n.SecretToken != "" && n.SecretToken == token {
				return true
			}
		}
	}
	return false
}

func verifyAgentSignature(token, timestamp, signature string) error {
	if timestamp == "" || signature == "" {
		return errors.New("子节点签名缺失 (缺少 1Panel-Token / 1Panel-Timestamp)")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("子节点签名时间戳格式错误")
	}
	skew := time.Now().Unix() - ts
	if skew > int64(MaxAgentClockSkew.Seconds()) || skew < -int64(MaxAgentClockSkew.Seconds()) {
		return errors.New("子节点签名时间戳超出允许窗口 (疑似重放请求)")
	}
	hash := md5.Sum([]byte("1panel" + token + timestamp))
	if hex.EncodeToString(hash[:]) != strings.ToLower(strings.TrimSpace(signature)) {
		return errors.New("子节点签名校验失败 (Signature Mismatch)")
	}
	return nil
}

// AgentAuth authenticates master->agent traffic: Bearer token plus the
// 1Panel-Token/1Panel-Timestamp signature pair (anti-replay), and records the
// forwarded master user for audit attribution.
func AgentAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if !MatchAgentToken(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "子节点通信密钥验证失败 (Secret Token Mismatch)",
			})
			return
		}

		if err := verifyAgentSignature(token, c.GetHeader("1Panel-Timestamp"), c.GetHeader("1Panel-Token")); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": err.Error(),
			})
			return
		}

		c.Set("authMethod", constant.AuthMethodJWT)
		c.Set("isAgentRequest", true)
		if forwarded := c.GetHeader("X-Panel-User"); forwarded != "" {
			if decoded, err := url.QueryUnescape(forwarded); err == nil {
				forwarded = decoded
			}
			c.Set(PanelForwardedUserKey, forwarded)
			if global.LOG != nil {
				global.LOG.Infof("[AgentAuth] master user %s -> %s %s", forwarded, c.Request.Method, c.Request.URL.Path)
			}
		}
		c.Next()
	}
}
