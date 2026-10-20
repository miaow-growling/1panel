package helper

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/1Panel-dev/1Panel/core/app/repo"
	"github.com/1Panel-dev/1Panel/core/global"
	"github.com/1Panel-dev/1Panel/core/init/proxy"
	"github.com/1Panel-dev/1Panel/core/utils/ssh"
	"github.com/gin-gonic/gin"
)

type multiNodeHelper struct{}

func NewIMultiNodeProvider() *multiNodeHelper {
	return &multiNodeHelper{}
}

var (
	remoteProxyCache = make(map[string]*httputil.ReverseProxy)
	remoteProxyMu    sync.RWMutex
)

// agentSignatureHeaders builds the Bearer + 1Panel-Token/1Panel-Timestamp pair
// the remote agent validates (mirrors agent-side certificate middleware).
func agentSignatureHeaders(token string) (string, string) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	hash := md5.Sum([]byte("1panel" + token + now))
	return now, hex.EncodeToString(hash[:])
}

func getOrCreateRemoteProxy(targetURL *url.URL) *httputil.ReverseProxy {
	key := targetURL.String()
	remoteProxyMu.RLock()
	p, ok := remoteProxyCache[key]
	remoteProxyMu.RUnlock()
	if ok {
		return p
	}
	remoteProxyMu.Lock()
	defer remoteProxyMu.Unlock()
	if p, ok := remoteProxyCache[key]; ok {
		return p
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = (&multiNodeHelper{}).LoadRequestTransport()
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		global.LOG.Errorf("[MultiNode] error proxying to %s: %v", targetURL.String(), err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"target node unreachable: ` + err.Error() + `"}`))
	}
	remoteProxyCache[key] = proxy
	return proxy
}

func loadRemoteNode(currentNode string) (*model.HostNode, error) {
	nodeRepo := repo.NewIHostNodeRepo()
	if id, err := strconv.ParseUint(currentNode, 10, 64); err == nil && id > 0 {
		if node, err := nodeRepo.Get(nodeRepo.WithByID(uint(id))); err == nil {
			return &node, nil
		}
	}
	if node, err := nodeRepo.Get(nodeRepo.WithByName(currentNode)); err == nil {
		return &node, nil
	}
	return nil, fmt.Errorf("node %q not found", currentNode)
}

// Proxy forwards the request to the remote worker agent selected via the
// CurrentNode header/query. Community implementation: HTTPS + token signature.
func (m *multiNodeHelper) Proxy(c *gin.Context, currentNode string) {
	if currentNode != "local" && currentNode != "" {
		node, err := loadRemoteNode(currentNode)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
			c.Abort()
			return
		}
		if node.Status != "online" {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "target node is offline"})
			c.Abort()
			return
		}
		remoteURL, err := url.Parse(node.Address)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "invalid node address: " + err.Error()})
			c.Abort()
			return
		}
		if global.LOG != nil {
			global.LOG.Infof("[MultiNode] forwarding %s %s to node %s (%s)", c.Request.Method, c.Request.URL.Path, node.Name, node.Address)
		}
		req := c.Request
		req.URL.Scheme = remoteURL.Scheme
		req.URL.Host = remoteURL.Host
		req.Host = remoteURL.Host

		forwardedUser := ""
		if ps, err := global.SESSION.Get(c); err == nil {
			forwardedUser = ps.Name
		}
		req.Header.Del("Cookie")
		req.Header.Del("X-Panel-User")
		ts, sig := agentSignatureHeaders(node.SecretToken)
		req.Header.Set("Authorization", "Bearer "+node.SecretToken)
		req.Header.Set("1Panel-Timestamp", ts)
		req.Header.Set("1Panel-Token", sig)
		if forwardedUser != "" {
			req.Header.Set("X-Panel-User", url.QueryEscape(forwardedUser))
		}

		defer func() {
			if err := recover(); err != nil && err != http.ErrAbortHandler {
				global.LOG.Debug(err)
			}
		}()
		getOrCreateRemoteProxy(remoteURL).ServeHTTP(c.Writer, req)
		c.Abort()
		return
	}
	defer func() {
		if err := recover(); err != nil && err != http.ErrAbortHandler {
			global.LOG.Debug(err)
		}
	}()
	proxy.LocalAgentProxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func (m *multiNodeHelper) ProxyDocker(proxyURL string) error { return nil }

func (m *multiNodeHelper) UpdateGroup(name string, group, newGroup uint) error { return nil }

func (m *multiNodeHelper) CheckBackupUsed(name string) error { return nil }

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

func (m *multiNodeHelper) LoadNodeInfo(currentNode string) (*ssh.ConnInfo, string, error) {
	return nil, "", nil
}

func (m *multiNodeHelper) Sync(dataType string) error { return nil }

func (m *multiNodeHelper) AutoUpgradeWithMaster() {}
