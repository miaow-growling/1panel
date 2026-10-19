package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/1Panel-dev/1Panel/backend/app/repo"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/gin-gonic/gin"
)

var (
	proxyCache      = make(map[string]*httputil.ReverseProxy)
	proxyMu         sync.RWMutex
	proxyTransport  = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
)

func getOrCreateProxy(targetURL *url.URL) *httputil.ReverseProxy {
	key := targetURL.String()
	proxyMu.RLock()
	p, ok := proxyCache[key]
	proxyMu.RUnlock()
	if ok {
		return p
	}

	proxyMu.Lock()
	defer proxyMu.Unlock()
	if p, ok := proxyCache[key]; ok {
		return p
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = proxyTransport
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		global.LOG.Errorf("[NodeProxy] error proxying to %s: %v", targetURL.String(), err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"目标节点连接失败: ` + err.Error() + `"}`))
	}
	proxyCache[key] = proxy
	return proxy
}

type safeProxyWriter struct {
	gin.ResponseWriter
}

func (s *safeProxyWriter) CloseNotify() (ch <-chan bool) {
	defer func() {
		if r := recover(); r != nil {
			ch = make(chan bool)
		}
	}()
	return s.ResponseWriter.CloseNotify()
}

// NodeProxyMiddleware transparently forwards requests destined for other nodes
func NodeProxyMiddleware() gin.HandlerFunc {
	nodeRepo := repo.NewINodeRepo()

	return func(c *gin.Context) {
		targetNode := strings.TrimSpace(c.GetHeader("X-Target-Node"))

		// If no target node or target is local, continue to local handlers
		if targetNode == "" || targetNode == "local" {
			c.Next()
			return
		}

		// Local management endpoints should never be proxied
		path := c.Request.URL.Path
		if path == "/api/v1/nodes" || strings.HasPrefix(path, "/api/v1/nodes/") || strings.HasPrefix(path, "/api/v1/auth/") {
			c.Next()
			return
		}

		// Verify that caller on Master is authenticated
		isAuth := false
		if method, exist := c.Get("authMethod"); exist && method == constant.AuthMethodJWT {
			isAuth = true
		} else if sId, err := c.Cookie(constant.SessionName); err == nil && sId != "" {
			if _, err := global.SESSION.Get(sId); err == nil {
				isAuth = true
			}
		} else if c.GetHeader("1Panel-Token") != "" {
			isAuth = true
		}

		if !isAuth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未登录或登录已超时",
			})
			return
		}

		// Find target node in database
		node, err := nodeRepo.Get(targetNode)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"code":    502,
				"message": "目标节点不存在",
			})
			return
		}

		if node.Status != "online" {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"code":    502,
				"message": "目标节点当前处于离线状态",
			})
			return
		}

		remoteURL, err := url.Parse(node.Address)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"code":    502,
				"message": "目标节点地址解析失败: " + err.Error(),
			})
			return
		}

		proxy := getOrCreateProxy(remoteURL)

		// Create dynamic director for this specific request
		req := c.Request
		req.URL.Scheme = remoteURL.Scheme
		req.URL.Host = remoteURL.Host
		req.Host = remoteURL.Host

		// Set authentication headers for agent
		req.Header.Set("Authorization", "Bearer "+node.SecretToken)
		req.Header.Del("X-Target-Node")

		// Transparently serve request
		proxy.ServeHTTP(&safeProxyWriter{ResponseWriter: c.Writer}, req)
		c.Abort()
	}
}
