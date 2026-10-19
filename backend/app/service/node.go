package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/backend/app/dto"
	"github.com/1Panel-dev/1Panel/backend/app/model"
	"github.com/1Panel-dev/1Panel/backend/app/repo"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/google/uuid"
)

type NodeService struct {
	nodeRepo repo.INodeRepo
}

type INodeService interface {
	Page(req dto.NodeSearch) (int64, []dto.NodeInfo, error)
	List() ([]dto.NodeInfo, error)
	Create(req dto.NodeCreate) error
	Update(req dto.NodeUpdate) error
	Delete(id string) error
	TestConnection(req dto.NodeTestReq) error
	GetNodeByID(id string) (model.HostNode, error)
}

func NewINodeService() INodeService {
	return &NodeService{
		nodeRepo: repo.NewINodeRepo(),
	}
}

func (s *NodeService) GetNodeByID(id string) (model.HostNode, error) {
	return s.nodeRepo.Get(id)
}

func (s *NodeService) List() ([]dto.NodeInfo, error) {
	nodes, err := s.nodeRepo.List()
	if err != nil {
		return nil, err
	}

	res := make([]dto.NodeInfo, 0, len(nodes)+1)
	// Master node is always the first node (local)
	res = append(res, dto.NodeInfo{
		ID:          "local",
		Name:        "Master 主节点 (当前服务器)",
		Address:     "127.0.0.1",
		Status:      "online",
		Description: "主控管理节点",
	})

	for _, node := range nodes {
		res = append(res, dto.NodeInfo{
			ID:          node.ID,
			Name:        node.Name,
			Address:     node.Address,
			Status:      node.Status,
			Description: node.Description,
			CreatedAt:   node.CreatedAt.Format(constant.DateTimeLayout),
			UpdatedAt:   node.UpdatedAt.Format(constant.DateTimeLayout),
		})
	}
	return res, nil
}

func (s *NodeService) Page(req dto.NodeSearch) (int64, []dto.NodeInfo, error) {
	total, nodes, err := s.nodeRepo.Page(req.Page, req.PageSize, req.Info)
	if err != nil {
		return 0, nil, err
	}

	items := make([]dto.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, dto.NodeInfo{
			ID:          node.ID,
			Name:        node.Name,
			Address:     node.Address,
			Status:      node.Status,
			Description: node.Description,
			CreatedAt:   node.CreatedAt.Format(constant.DateTimeLayout),
			UpdatedAt:   node.UpdatedAt.Format(constant.DateTimeLayout),
		})
	}
	return total, items, nil
}

func (s *NodeService) Create(req dto.NodeCreate) error {
	addr := strings.TrimRight(strings.TrimSpace(req.Address), "/")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	// Test connection before creating
	if err := s.TestConnection(dto.NodeTestReq{
		Address:     addr,
		SecretToken: req.SecretToken,
	}); err != nil {
		return fmt.Errorf("无法连接目标节点: %w", err)
	}

	node := model.HostNode{
		ID:          "node-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12],
		Name:        strings.TrimSpace(req.Name),
		Address:     addr,
		SecretToken: strings.TrimSpace(req.SecretToken),
		Status:      "online",
		Description: req.Description,
	}

	return s.nodeRepo.Create(&node)
}

func (s *NodeService) Update(req dto.NodeUpdate) error {
	addr := strings.TrimRight(strings.TrimSpace(req.Address), "/")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	node, err := s.nodeRepo.Get(req.ID)
	if err != nil {
		return errors.New("目标节点不存在")
	}

	node.Name = strings.TrimSpace(req.Name)
	node.Address = addr
	node.Description = req.Description
	if strings.TrimSpace(req.SecretToken) != "" {
		node.SecretToken = strings.TrimSpace(req.SecretToken)
	}

	return s.nodeRepo.Update(&node)
}

func (s *NodeService) Delete(id string) error {
	if id == "local" || id == "" {
		return errors.New("cannot delete local master node")
	}
	return s.nodeRepo.Delete(id)
}

func (s *NodeService) TestConnection(req dto.NodeTestReq) error {
	addr := strings.TrimRight(strings.TrimSpace(req.Address), "/")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	testURL := addr + "/health"
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	httpReq, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(req.SecretToken))

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("网络连通失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("节点响应状态异常: HTTP %d", resp.StatusCode)
	}
	return nil
}
