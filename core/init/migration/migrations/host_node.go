package migrations

import (
	"github.com/1Panel-dev/1Panel/core/app/model"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var AddHostNodes = &gormigrate.Migration{
	ID:      "20261020-host-nodes",
	Migrate: func(tx *gorm.DB) error { return tx.AutoMigrate(&model.HostNode{}) },
}
