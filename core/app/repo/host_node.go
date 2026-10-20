package repo

import (
	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/1Panel-dev/1Panel/core/global"
	"gorm.io/gorm"
)

type HostNodeRepo struct{}

type IHostNodeRepo interface {
	GetList(opts ...global.DBOption) ([]model.HostNode, error)
	Get(opts ...global.DBOption) (model.HostNode, error)
	Create(node *model.HostNode) error
	Update(node *model.HostNode) error
	Delete(opts ...global.DBOption) error

	WithByID(id uint) global.DBOption
	WithByName(name string) global.DBOption
}

func NewIHostNodeRepo() IHostNodeRepo {
	return &HostNodeRepo{}
}

func (h *HostNodeRepo) WithByID(id uint) global.DBOption {
	return func(g *gorm.DB) *gorm.DB { return g.Where("id = ?", id) }
}

func (h *HostNodeRepo) WithByName(name string) global.DBOption {
	return func(g *gorm.DB) *gorm.DB { return g.Where("name = ?", name) }
}

func (h *HostNodeRepo) GetList(opts ...global.DBOption) ([]model.HostNode, error) {
	var nodes []model.HostNode
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&nodes).Error
	return nodes, err
}

func (h *HostNodeRepo) Get(opts ...global.DBOption) (model.HostNode, error) {
	var node model.HostNode
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&node).Error
	return node, err
}

func (h *HostNodeRepo) Create(node *model.HostNode) error {
	return global.DB.Create(node).Error
}

func (h *HostNodeRepo) Update(node *model.HostNode) error {
	return global.DB.Save(node).Error
}

func (h *HostNodeRepo) Delete(opts ...global.DBOption) error {
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.HostNode{}).Error
}
