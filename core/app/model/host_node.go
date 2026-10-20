package model

import "time"

// HostNode stores remote worker agents registered on the master core.
type HostNode struct {
	BaseModel
	Name          string    `json:"name"`
	Address       string    `json:"address"` // https://host:port of the remote agent
	SecretToken   string    `json:"secretToken"`
	Status        string    `json:"status"`
	Version       string    `json:"version"`
	Description   string    `json:"description"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
}

func (h *HostNode) TableName() string { return "host_nodes" }
