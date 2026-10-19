package v1

import (
	"github.com/1Panel-dev/1Panel/backend/app/api/v1/helper"
	"github.com/1Panel-dev/1Panel/backend/app/dto"
	"github.com/1Panel-dev/1Panel/backend/constant"
	"github.com/gin-gonic/gin"
)

// @Tags Node
// @Summary List all nodes
// @Accept json
// @Success 200 {object} []dto.NodeInfo
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes [get]
func (b *BaseApi) ListNode(c *gin.Context) {
	nodes, err := nodeService.List()
	if err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, nodes)
}

// @Tags Node
// @Summary Search nodes by page
// @Accept json
// @Param request body dto.NodeSearch true "request"
// @Success 200 {object} dto.PageResult
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes/search [post]
func (b *BaseApi) SearchNode(c *gin.Context) {
	var req dto.NodeSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	total, items, err := nodeService.Page(req)
	if err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: items,
	})
}

// @Tags Node
// @Summary Create a new node
// @Accept json
// @Param request body dto.NodeCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes [post]
// @x-panel-log {"bodyKeys":["name","address"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"添加节点 [name][address]","formatEN":"create node [name][address]"}
func (b *BaseApi) CreateNode(c *gin.Context) {
	var req dto.NodeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := nodeService.Create(req); err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

// @Tags Node
// @Summary Update node info
// @Accept json
// @Param request body dto.NodeUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes/update [post]
// @x-panel-log {"bodyKeys":["name","address"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新节点 [name][address]","formatEN":"update node [name][address]"}
func (b *BaseApi) UpdateNode(c *gin.Context) {
	var req dto.NodeUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := nodeService.Update(req); err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

// @Tags Node
// @Summary Delete node
// @Accept json
// @Param request body dto.NodeOperateByID true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes/del [post]
// @x-panel-log {"bodyKeys":["id"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除节点 [id]","formatEN":"delete node [id]"}
func (b *BaseApi) DeleteNode(c *gin.Context) {
	var req dto.NodeOperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := nodeService.Delete(req.ID); err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

// @Tags Node
// @Summary Test node connection
// @Accept json
// @Param request body dto.NodeTestReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /nodes/test [post]
func (b *BaseApi) TestNode(c *gin.Context) {
	var req dto.NodeTestReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := nodeService.TestConnection(req); err != nil {
		helper.ErrorWithDetail(c, constant.CodeErrInternalServer, constant.ErrTypeInternalServer, err)
		return
	}
	helper.SuccessWithData(c, nil)
}
