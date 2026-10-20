package service

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/1Panel-dev/1Panel/core/app/repo"
	"github.com/1Panel-dev/1Panel/core/global"
)

type HostNodeService struct {
	nodeRepo repo.IHostNodeRepo
}

// One-time download tokens for the agent binary installer script.
var (
	downloadTokens   = make(map[string]time.Time)
	downloadTokensMu sync.Mutex
)

const downloadTokenTTL = 10 * time.Minute

func CreateDownloadToken() string {
	token := strings.ReplaceAll(uuid.New().String(), "-", "")
	downloadTokensMu.Lock()
	defer downloadTokensMu.Unlock()
	downloadTokens[token] = time.Now().Add(downloadTokenTTL)
	return token
}

func VerifyDownloadToken(token string) bool {
	downloadTokensMu.Lock()
	defer downloadTokensMu.Unlock()
	exp, ok := downloadTokens[token]
	if !ok {
		return false
	}
	delete(downloadTokens, token)
	return time.Now().Before(exp)
}

func AgentBinaryPath() string {
	for _, p := range []string{"/usr/local/bin/1panel-agent", "./1panel-agent", "1panel-agent"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

type IHostNodeService interface {
	List() ([]dto.HostNodeInfo, error)
	ListForFrontend() ([]dto.FrontendNodeItem, error)
	SimpleAll() ([]dto.SimpleNodeItem, error)
	Create(req dto.HostNodeCreate) error
	Update(req dto.HostNodeUpdate) error
	Delete(id uint) error
	TestConnection(req dto.HostNodeTestReq) error
	TestConnectionByID(id uint) error
	BuildInstallScript(req dto.HostNodeInstallScriptReq, masterBaseURL string) (string, error)
	GetLocalAgentToken() (string, error)
}

func NewIHostNodeService() IHostNodeService {
	return &HostNodeService{nodeRepo: repo.NewIHostNodeRepo()}
}

func (s *HostNodeService) GetLocalAgentToken() (string, error) {
	if envToken := os.Getenv("PANEL_AGENT_TOKEN"); envToken != "" {
		return envToken, nil
	}
	settingRepo := repo.NewISettingRepo()
	if value, err := settingRepo.GetValueByKey("AgentToken"); err == nil && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("agent token not configured")
}

func (s *HostNodeService) List() ([]dto.HostNodeInfo, error) {
	nodes, err := s.nodeRepo.GetList()
	if err != nil {
		return nil, err
	}
	s.probeInstallingNodes(nodes)
	res := make([]dto.HostNodeInfo, 0, len(nodes)+1)
	res = append(res, dto.HostNodeInfo{
		ID:          0,
		Name:        "local",
		Address:     "127.0.0.1",
		Status:      "online",
		Description: "Master node (local)",
		IsLocal:     true,
	})
	for _, node := range nodes {
		res = append(res, dto.HostNodeInfo{
			ID:          node.ID,
			Name:        node.Name,
			Address:     node.Address,
			Status:      node.Status,
			Version:     node.Version,
			Description: node.Description,
			IsLocal:     false,
		})
	}
	return res, nil
}

func (s *HostNodeService) ListForFrontend() ([]dto.FrontendNodeItem, error) {
	nodes, err := s.nodeRepo.GetList()
	if err != nil {
		return nil, err
	}
	version := global.CONF.Base.Version
	res := []dto.FrontendNodeItem{{
		ID: 0, Name: "local", Addr: "127.0.0.1", Status: "Healthy",
		Version: version, IsXpack: false, IsBound: true,
	}}
	for _, node := range nodes {
		status := "Healthy"
		if node.Status != "online" {
			status = node.Status
		}
		nodeVersion := node.Version
		if nodeVersion == "" {
			nodeVersion = version
		}
		res = append(res, dto.FrontendNodeItem{
			ID: node.ID, Name: node.Name, Addr: node.Address, Status: status,
			Version: nodeVersion, IsXpack: false, IsBound: true,
		})
	}
	return res, nil
}

func (s *HostNodeService) SimpleAll() ([]dto.SimpleNodeItem, error) {
	nodes, err := s.nodeRepo.GetList()
	if err != nil {
		return nil, err
	}
	res := []dto.SimpleNodeItem{{
		ID: 0, Name: "local", Addr: "127.0.0.1", Description: "Master node (local)",
	}}
	for _, node := range nodes {
		res = append(res, dto.SimpleNodeItem{
			ID: node.ID, Name: node.Name, Addr: node.Address, Description: node.Description,
		})
	}
	return res, nil
}

func normalizeAddr(addr string) string {
	addr = strings.TrimRight(strings.TrimSpace(addr), "/")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "https://" + addr
	}
	return addr
}

func (s *HostNodeService) Create(req dto.HostNodeCreate) error {
	addr := normalizeAddr(req.Address)
	if err := s.TestConnection(dto.HostNodeTestReq{Address: addr, SecretToken: req.SecretToken}); err != nil {
		return fmt.Errorf("cannot reach target node: %w", err)
	}
	node := model.HostNode{
		Name:        strings.TrimSpace(req.Name),
		Address:     addr,
		SecretToken: strings.TrimSpace(req.SecretToken),
		Status:      "online",
		Version:     req.Version,
		Description: req.Description,
	}
	return s.nodeRepo.Create(&node)
}

func (s *HostNodeService) Update(req dto.HostNodeUpdate) error {
	node, err := s.nodeRepo.Get(s.nodeRepo.WithByID(req.ID))
	if err != nil {
		return fmt.Errorf("node not found")
	}
	node.Name = strings.TrimSpace(req.Name)
	node.Address = normalizeAddr(req.Address)
	node.Description = req.Description
	if strings.TrimSpace(req.SecretToken) != "" {
		node.SecretToken = strings.TrimSpace(req.SecretToken)
	}
	return s.nodeRepo.Update(&node)
}

func (s *HostNodeService) Delete(id uint) error {
	return s.nodeRepo.Delete(s.nodeRepo.WithByID(id))
}

func (s *HostNodeService) TestConnection(req dto.HostNodeTestReq) error {
	addr := normalizeAddr(req.Address)
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	httpReq, err := http.NewRequest(http.MethodGet, addr+"/health", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.SecretToken))
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("network unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("node responded with HTTP %d (secret token mismatch?)", resp.StatusCode)
	}
	return nil
}

func (s *HostNodeService) TestConnectionByID(id uint) error {
	node, err := s.nodeRepo.Get(s.nodeRepo.WithByID(id))
	if err != nil {
		return fmt.Errorf("node not found")
	}
	return s.TestConnection(dto.HostNodeTestReq{Address: node.Address, SecretToken: node.SecretToken})
}

// BuildInstallScript renders a self-contained bash script that downloads the
// agent binary from this master, provisions 1pctl params and starts the node.
// BuildInstallScript pre-registers the node (status=installing) and renders
// the installer script, so the node shows up the moment the command is
// generated and flips to online automatically once the agent boots.
func (s *HostNodeService) BuildInstallScript(req dto.HostNodeInstallScriptReq, masterBaseURL string) (string, error) {
	binaryPath := AgentBinaryPath()
	if binaryPath == "" {
		return "", fmt.Errorf("agent binary not found on master")
	}
	token := CreateDownloadToken()
	port := req.Port
	if port == 0 {
		port = 9998
	}
	secret := strings.TrimSpace(req.SecretToken)
	if secret == "" {
		return "", fmt.Errorf("secret token is required")
	}

	// Pre-register: address uses the user-supplied node IP so the master can
	// reach the agent once it boots behind the cloud firewall.
	nodeIP := strings.TrimSpace(req.NodeIP)
	if nodeIP == "" {
		return "", fmt.Errorf("node IP is required for auto registration")
	}
	addr := fmt.Sprintf("https://%s:%d", nodeIP, port)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("node-%s", strings.ReplaceAll(nodeIP, ".", "-"))
	}
	existing, err := s.nodeRepo.Get(s.nodeRepo.WithByName(name))
	if err == nil {
		existing.Address = addr
		existing.SecretToken = secret
		existing.Status = "installing"
		if err := s.nodeRepo.Update(&existing); err != nil {
			return "", err
		}
	} else {
		if err := s.nodeRepo.Create(&model.HostNode{
			Name:        name,
			Address:     addr,
			SecretToken: secret,
			Status:      "installing",
			Description: "一键安装节点（等待 Agent 上线）",
		}); err != nil {
			return "", err
		}
	}
	script := fmt.Sprintf(`#!/bin/bash
# 1Panel 社区多节点 —— 被控端一键安装脚本
# 由主控生成；在目标节点机器上以 root 执行
set -e

AGENT_PORT=%d
AGENT_TOKEN=%q
MASTER=%q
DOWNLOAD_TOKEN=%q

echo "==> 下载 1panel-agent (来自主控)"
if command -v curl >/dev/null 2>&1; then
    curl -skf "$MASTER/api/v2/core/nodes/agent-binary?downloadToken=$DOWNLOAD_TOKEN" -o /usr/local/bin/1panel-agent
elif command -v wget >/dev/null 2>&1; then
    wget --no-check-certificate -qO /usr/local/bin/1panel-agent "$MASTER/api/v2/core/nodes/agent-binary?downloadToken=$DOWNLOAD_TOKEN"
else
    echo "需要 curl 或 wget 之一" && exit 1
fi
chmod +x /usr/local/bin/1panel-agent

echo "==> 写入参数文件 /usr/local/bin/1pctl"
cat > /usr/local/bin/1pctl <<'PCTL'
BASE_DIR=/opt
ORIGINAL_PORT=9999
ORIGINAL_VERSION=v2-node
ORIGINAL_USERNAME=admin
ORIGINAL_PASSWORD=admin123
ORIGINAL_ENTRANCE=entrance
LANGUAGE=zh
PCTL
chmod +x /usr/local/bin/1pctl

mkdir -p /opt

if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    echo "==> 注册 systemd 服务 (1panel-agent-node)"
    cat > /etc/systemd/system/1panel-agent-node.service <<'UNIT'
[Unit]
Description=1Panel Agent (node mode)
After=network.target

[Service]
Environment=PANEL_NODE_MODE=true
Environment=PANEL_NODE_PORT=${AGENT_PORT}
Environment=PANEL_AGENT_TOKEN=${AGENT_TOKEN}
WorkingDirectory=/opt
ExecStart=/usr/local/bin/1panel-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT
    systemctl daemon-reload
    systemctl enable --now 1panel-agent-node
else
    echo "==> systemd 不可用,使用 nohup 后台启动"
    pkill -f "1panel-agent" 2>/dev/null || true
    PANEL_NODE_MODE=true PANEL_NODE_PORT=$AGENT_PORT PANEL_AGENT_TOKEN=$AGENT_TOKEN \
      nohup /usr/local/bin/1panel-agent > /var/log/1panel-agent-node.log 2>&1 &
fi

echo "==> 完成。请在主控『集群节点』页面添加本节点："
echo "    地址: https://<本机IP>:$AGENT_PORT   密钥: 即 AGENT_TOKEN"
`, port, secret, masterBaseURL, token)
	return script, nil
}

// probeInstallingNodes checks pending nodes with a short /health probe and
// flips them to online once the freshly installed agent answers.
func (s *HostNodeService) probeInstallingNodes(nodes []model.HostNode) {
	for i := range nodes {
		if nodes[i].Status != "installing" {
			continue
		}
		client := &http.Client{
			Timeout:   3 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		}
		req, err := http.NewRequest(http.MethodGet, strings.TrimRight(nodes[i].Address, "/")+"/health", nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+nodes[i].SecretToken)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			nodes[i].Status = "online"
			_ = s.nodeRepo.Update(&nodes[i])
		}
	}
}
