package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/module"
)

type createModuleReq struct {
	Name string `json:"name" binding:"required"`
}

func (s *Server) handleListModules(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireMembership(c, projectID) {
		return
	}
	list, err := s.store.ListModules(projectID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载模块失败")
		return
	}
	if list == nil {
		list = []module.Module{}
	}
	OK(c, list)
}

// handleCreateModule upserts a tag-like module by name（即用即写）.
func (s *Server) handleCreateModule(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireWrite(c, projectID) {
		return
	}
	var req createModuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写模块名称")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "模块名称不能为空")
		return
	}
	md, err := s.store.EnsureModule(projectID, name)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "创建模块失败")
		return
	}
	OK(c, md)
}

func (s *Server) handleDeleteModule(c *gin.Context) {
	projectID := c.Param("id")
	moduleID := c.Param("moduleId")
	if !s.requireWrite(c, projectID) {
		return
	}
	md, err := s.store.GetModule(moduleID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "模块不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载模块失败")
		return
	}
	if md.ProjectID != projectID {
		Fail(c, http.StatusNotFound, "模块不存在")
		return
	}
	if err := s.store.DeleteModule(moduleID); err != nil {
		Fail(c, http.StatusInternalServerError, "删除模块失败")
		return
	}
	OK(c, gin.H{"id": moduleID})
}
