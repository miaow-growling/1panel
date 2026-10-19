package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/1Panel-dev/1Panel/backend/app/model"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.HostNode{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	global.DB = db
	return db
}

func TestNodeProxy_LocalPassthrough(t *testing.T) {
	setupTestDB(t)

	r := gin.New()
	r.Use(NodeProxyMiddleware())
	r.GET("/api/v1/containers/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "local_success"})
	})

	// Case 1: no X-Target-Node header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Case 2: X-Target-Node is local
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/containers/search", nil)
	req2.Header.Set("X-Target-Node", "local")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

func TestNodeProxy_UnauthenticatedTarget(t *testing.T) {
	setupTestDB(t)

	r := gin.New()
	r.Use(NodeProxyMiddleware())
	r.GET("/api/v1/containers/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers/search", nil)
	req.Header.Set("X-Target-Node", "node-remote-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized when not logged in, got %d", w.Code)
	}
}

func TestNodeProxy_RemoteForwarding(t *testing.T) {
	db := setupTestDB(t)

	// Spin up a mock sub-node Agent HTTP server
	receivedAuth := ""
	receivedTargetNode := ""
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedTargetNode = r.Header.Get("X-Target-Node")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"result": "from_agent_node",
			"path":   r.URL.Path,
		})
	}))
	defer mockAgent.Close()

	// Register sub-node in DB
	node := model.HostNode{
		ID:          "node-test-1",
		Name:        "Test Worker Node",
		Address:     mockAgent.URL,
		SecretToken: "secret-token-xyz-123",
		Status:      "online",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("failed to insert test node: %v", err)
	}

	r := gin.New()
	// Mock authenticated user on Master
	r.Use(func(c *gin.Context) {
		c.Set("authMethod", constant.AuthMethodJWT)
		c.Next()
	})
	r.Use(NodeProxyMiddleware())
	r.GET("/api/v1/containers/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "local"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers/search?page=1", nil)
	req.Header.Set("X-Target-Node", "node-test-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	if receivedAuth != "Bearer secret-token-xyz-123" {
		t.Fatalf("expected Authorization Bearer secret-token-xyz-123, got %s", receivedAuth)
	}

	if receivedTargetNode != "" {
		t.Fatalf("expected X-Target-Node to be stripped, got %s", receivedTargetNode)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if resp["result"] != "from_agent_node" {
		t.Fatalf("expected from_agent_node, got %v", resp["result"])
	}
}

func TestAgentAuth(t *testing.T) {
	_ = os.Setenv("PANEL_AGENT_TOKEN", "agent-secret-key-999")
	defer os.Unsetenv("PANEL_AGENT_TOKEN")

	r := gin.New()
	r.Use(AgentAuth())
	r.GET("/test-auth", func(c *gin.Context) {
		authMethod, _ := c.Get("authMethod")
		isAgent, _ := c.Get("isAgentRequest")
		c.JSON(http.StatusOK, gin.H{
			"authMethod": authMethod,
			"isAgent":    isAgent,
		})
	})

	// Valid agent token
	req := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	req.Header.Set("Authorization", "Bearer agent-secret-key-999")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["authMethod"] != constant.AuthMethodJWT {
		t.Fatalf("expected authMethod %v, got %v", constant.AuthMethodJWT, resp["authMethod"])
	}
	if resp["isAgent"] != true {
		t.Fatalf("expected isAgent true, got %v", resp["isAgent"])
	}

	// Invalid agent token
	reqInvalid := httptest.NewRequest(http.MethodGet, "/test-auth", nil)
	reqInvalid.Header.Set("Authorization", "Bearer wrong-token")
	wInvalid := httptest.NewRecorder()
	r.ServeHTTP(wInvalid, reqInvalid)

	var respInvalid map[string]interface{}
	_ = json.Unmarshal(wInvalid.Body.Bytes(), &respInvalid)
	if respInvalid["authMethod"] != nil {
		t.Fatalf("expected authMethod nil for wrong token, got %v", respInvalid["authMethod"])
	}
}
