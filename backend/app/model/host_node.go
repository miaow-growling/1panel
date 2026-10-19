package model

import "time"

type HostNode struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Address     string    `gorm:"size:255;not null" json:"address"` // e.g. http://10.x.x.x:8081 or https://node.domain.com
	SecretToken string    `gorm:"size:255;not null" json:"secretToken,omitempty"` // Token for Master-Agent authentication
	Status      string    `gorm:"size:32;default:'online'" json:"status"`           // online / offline
	Description string    `gorm:"size:256" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (h HostNode) TableName() string {
	return "host_nodes"
}
