package repo

import (
	"strings"

	"github.com/1Panel-dev/1Panel/backend/app/model"
	"github.com/1Panel-dev/1Panel/backend/global"
)

type NodeRepo struct{}

type INodeRepo interface {
	Get(id string) (model.HostNode, error)
	List() ([]model.HostNode, error)
	Page(page, size int, info string) (int64, []model.HostNode, error)
	Create(node *model.HostNode) error
	Update(node *model.HostNode) error
	Delete(id string) error
	UpdateStatus(id string, status string) error
}

func NewINodeRepo() INodeRepo {
	return &NodeRepo{}
}

func (n *NodeRepo) Get(id string) (model.HostNode, error) {
	var node model.HostNode
	err := global.DB.Where("id = ?", id).First(&node).Error
	return node, err
}

func (n *NodeRepo) List() ([]model.HostNode, error) {
	var nodes []model.HostNode
	err := global.DB.Order("created_at asc").Find(&nodes).Error
	return nodes, err
}

func (n *NodeRepo) Page(page, size int, info string) (int64, []model.HostNode, error) {
	var nodes []model.HostNode
	db := global.DB.Model(&model.HostNode{})
	if strings.TrimSpace(info) != "" {
		like := "%" + strings.TrimSpace(info) + "%"
		db = db.Where("name LIKE ? OR address LIKE ? OR description LIKE ?", like, like, like)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, nil, err
	}
	err := db.Order("created_at desc").Limit(size).Offset(size * (page - 1)).Find(&nodes).Error
	return count, nodes, err
}

func (n *NodeRepo) Create(node *model.HostNode) error {
	return global.DB.Create(node).Error
}

func (n *NodeRepo) Update(node *model.HostNode) error {
	updates := map[string]interface{}{
		"name":        node.Name,
		"address":     node.Address,
		"description": node.Description,
		"status":      node.Status,
	}
	if node.SecretToken != "" {
		updates["secret_token"] = node.SecretToken
	}
	return global.DB.Model(&model.HostNode{}).Where("id = ?", node.ID).Updates(updates).Error
}

func (n *NodeRepo) Delete(id string) error {
	return global.DB.Where("id = ?", id).Delete(&model.HostNode{}).Error
}

func (n *NodeRepo) UpdateStatus(id string, status string) error {
	return global.DB.Model(&model.HostNode{}).Where("id = ?", id).Update("status", status).Error
}
