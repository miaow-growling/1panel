package helper

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/common"
	"github.com/1Panel-dev/1Panel/agent/utils/xpack/providers"
	"github.com/gin-gonic/gin"
)

type multiNodeHelper struct{}

func NewIMultiNodeProvider() providers.MultiNodeProvider {
	return &multiNodeHelper{}
}

func (m *multiNodeHelper) RemoveTamper(website string) {}

func (m *multiNodeHelper) StartClam(startClam *model.Clam, isUpdate bool) (int, error) {
	return 0, buserr.New("ErrXpackNotFound")
}

func (m *multiNodeHelper) LoadNodeInfo(isBase bool) (model.NodeInfo, error) {
	var info model.NodeInfo
	info.BaseDir = common.LoadParams("BASE_DIR")
	info.Version = common.LoadParams("ORIGINAL_VERSION")
	// Community node mode: run this agent as a remote worker for a master
	// core. Enable with PANEL_NODE_MODE=true; token via PANEL_AGENT_TOKEN.
	if os.Getenv("PANEL_NODE_MODE") == "true" {
		info.Scope = "node"
		global.IsMaster = false
		if port := os.Getenv("PANEL_NODE_PORT"); port != "" {
			if p, err := strconv.Atoi(port); err == nil && p > 0 {
				info.NodePort = uint(p)
			}
		}
		if info.NodePort == 0 {
			info.NodePort = 9998
		}
		return info, nil
	}
	info.Scope = "master"
	global.IsMaster = true
	return info, nil
}

func (m *multiNodeHelper) GetImagePrefix() string {
	return ""
}

func (m *multiNodeHelper) IsUseCustomApp() bool {
	return false
}

func (m *multiNodeHelper) IsXpack() bool {
	return false
}

func (m *multiNodeHelper) LoadRequestTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       15 * time.Second,
	}
}

// ValidateCertificate authenticates master->node requests in node mode:
// Bearer token plus 1Panel-Token/1Panel-Timestamp signature (anti-replay).
func (m *multiNodeHelper) ValidateCertificate(c *gin.Context) bool {
	if global.IsMaster {
		return true
	}
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("missing bearer token"))
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	expected := os.Getenv("PANEL_AGENT_TOKEN")
	if expected == "" {
		if value, err := repo.NewISettingRepo().GetValueByKey("AgentToken"); err == nil {
			expected = value
		}
	}
	if expected == "" || token != expected {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("secret token mismatch"))
		return false
	}
	ts := c.GetHeader("1Panel-Timestamp")
	sig := c.GetHeader("1Panel-Token")
	if ts == "" || sig == "" {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("missing 1Panel-Token/1Panel-Timestamp signature"))
		return false
	}
	tsVal, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("invalid signature timestamp"))
		return false
	}
	if skew := time.Now().Unix() - tsVal; skew > 300 || skew < -300 {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("signature timestamp outside allowed window"))
		return false
	}
	hash := md5.Sum([]byte("1panel" + token + ts))
	if hex.EncodeToString(hash[:]) != strings.ToLower(strings.TrimSpace(sig)) {
		_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("signature mismatch"))
		return false
	}
	if user := c.GetHeader("X-Panel-User"); user != "" {
		if decoded, err := url.QueryUnescape(user); err == nil {
			user = decoded
		}
		global.LOG.Infof("[NodeAgent] master user %s -> %s %s", user, c.Request.Method, c.Request.URL.Path)
	}
	return true
}

func (m *multiNodeHelper) PushSSLToNode(websiteSSL *model.WebsiteSSL) error {
	return nil
}

func (m *multiNodeHelper) GetAgentInfo() (*dto.AgentInfo, error) {
	return nil, nil
}
