package dto

type HostNodeInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Status      string `json:"status"`
	Version     string `json:"version"`
	Description string `json:"description"`
	IsLocal     bool   `json:"isLocal"`
}

// FrontendNodeItem matches Setting.NodeItem expected by the v2 frontend.
type FrontendNodeItem struct {
	ID       uint   `json:"id"`
	GroupID  uint   `json:"groupID"`
	Addr     string `json:"addr"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	IsXpack  bool   `json:"isXpack"`
	IsBound  bool   `json:"isBound"`
	IsFavorite bool `json:"isFavorite"`
	Name     string `json:"name"`
}

// SimpleNodeItem matches Setting.SimpleNodeItem expected by the v2 frontend.
type SimpleNodeItem struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Addr             string `json:"addr"`
	Description      string `json:"description"`
	SystemVersion    string `json:"systemVersion"`
	SecurityEntrance string `json:"securityEntrance"`
	CpuUsedPercent   float64 `json:"cpuUsedPercent"`
	CpuTotal         int    `json:"cpuTotal"`
	MemoryTotal      uint64 `json:"memoryTotal"`
	MemoryUsedPercent float64 `json:"memoryUsedPercent"`
}

type HostNodeCreate struct {
	Name        string `json:"name" validate:"required"`
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken" validate:"required"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type HostNodeUpdate struct {
	ID          uint   `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken"`
	Description string `json:"description"`
}

type HostNodeTestReq struct {
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken" validate:"required"`
}

type HostNodeTestByID struct {
	ID uint `json:"id" validate:"required"`
}

type HostNodeInstallScriptReq struct {
	Port        uint   `json:"port"`
	SecretToken string `json:"secretToken" validate:"required"`
}
