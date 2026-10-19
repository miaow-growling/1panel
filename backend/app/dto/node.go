package dto

type NodeCreate struct {
	Name        string `json:"name" validate:"required"`
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken" validate:"required"`
	Description string `json:"description"`
}

type NodeUpdate struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken"`
	Description string `json:"description"`
}

type NodeSearch struct {
	PageInfo
	Info string `json:"info"`
}

type NodeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	SecretToken string `json:"secretToken,omitempty"`
	Status      string `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type NodeTestReq struct {
	Address     string `json:"address" validate:"required"`
	SecretToken string `json:"secretToken" validate:"required"`
}

type NodeOperateByID struct {
	ID string `json:"id" validate:"required"`
}
