package v2

import (
	"fmt"
	"net/http"

	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/app/service"
	"github.com/gin-gonic/gin"
)

// @Tags Host Node
// @Summary List registered nodes (management view)
// @Success 200 {object} []dto.HostNodeInfo
// @Router /nodes [get]
func (b *BaseApi) ListHostNodes(c *gin.Context) {
	nodes, err := hostNodeService.List()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, nodes)
}

// @Tags Host Node
// @Summary List nodes in the frontend NodeItem shape
// @Router /nodes/list [post]
func (b *BaseApi) ListHostNodesForFrontend(c *gin.Context) {
	nodes, err := hostNodeService.ListForFrontend()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, nodes)
}

// @Tags Host Node
// @Summary Simple node list for the frontend
// @Router /nodes/simple/all [get]
func (b *BaseApi) GetHostNodesSimple(c *gin.Context) {
	nodes, err := hostNodeService.SimpleAll()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, nodes)
}

// @Tags Host Node
// @Summary Create node
// @Param request body dto.HostNodeCreate true "request"
// @Router /nodes [post]
func (b *BaseApi) CreateHostNode(c *gin.Context) {
	var req dto.HostNodeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := hostNodeService.Create(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Host Node
// @Summary Update node
// @Param request body dto.HostNodeUpdate true "request"
// @Router /nodes/update [post]
func (b *BaseApi) UpdateHostNode(c *gin.Context) {
	var req dto.HostNodeUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := hostNodeService.Update(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Host Node
// @Summary Delete node
// @Param request body dto.OperateByID true "request"
// @Router /nodes/del [post]
func (b *BaseApi) DeleteHostNode(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := hostNodeService.Delete(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Host Node
// @Summary Test node connectivity and secret
// @Param request body dto.HostNodeTestReq true "request"
// @Router /nodes/test [post]
func (b *BaseApi) TestHostNode(c *gin.Context) {
	var req dto.HostNodeTestReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := hostNodeService.TestConnection(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Host Node
// @Summary Test node connectivity by id (uses stored secret)
// @Param request body dto.HostNodeTestByID true "request"
// @Router /nodes/test/id [post]
func (b *BaseApi) TestHostNodeByID(c *gin.Context) {
	var req dto.HostNodeTestByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := hostNodeService.TestConnectionByID(req.ID); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Host Node
// @Summary Generate one-click node installer script
// @Param request body dto.HostNodeInstallScriptReq true "request"
// @Router /nodes/install-script [post]
func (b *BaseApi) GetHostNodeInstallScript(c *gin.Context) {
	var req dto.HostNodeInstallScriptReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	masterBase := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	script, err := hostNodeService.BuildInstallScript(req, masterBase)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, script)
}

// @Tags Host Node
// @Summary Download agent binary with one-time token
// @Router /nodes/agent-binary [get]
func (b *BaseApi) DownloadAgentBinary(c *gin.Context) {
	if !service.VerifyDownloadToken(c.Query("downloadToken")) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "download token invalid or expired"})
		return
	}
	path := service.AgentBinaryPath()
	if path == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "agent binary not found on master"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=1panel-agent")
	c.File(path)
}
