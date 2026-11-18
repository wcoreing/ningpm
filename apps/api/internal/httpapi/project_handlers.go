package httpapi

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/project"
)

var projectKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

type createProjectReq struct {
	Key         string `json:"key" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (s *Server) handleListProjects(c *gin.Context) {
	list, err := s.store.ListProjects(currentUserID(c))
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载项目失败")
		return
	}
	if list == nil {
		list = []project.Project{}
	}
	OK(c, list)
}

func (s *Server) handleCreateProject(c *gin.Context) {
	var req createProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写项目标识和名称")
		return
	}
	key := strings.ToUpper(strings.TrimSpace(req.Key))
	name := strings.TrimSpace(req.Name)
	if !projectKeyRe.MatchString(key) {
		Fail(c, http.StatusBadRequest, "项目标识需为 2-10 位大写字母数字，且以字母开头")
		return
	}
	if name == "" {
		Fail(c, http.StatusBadRequest, "项目名称不能为空")
		return
	}
	if _, err := s.store.GetProjectByKey(key); err == nil {
		Fail(c, http.StatusConflict, "项目标识已存在")
		return
	} else if err != sql.ErrNoRows {
		Fail(c, http.StatusInternalServerError, "创建项目失败")
		return
	}

	p := &project.Project{
		Key:         key,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		CreatorID:   currentUserID(c),
	}
	if err := s.store.CreateProject(p); err != nil {
		Fail(c, http.StatusInternalServerError, "创建项目失败")
		return
	}
	OK(c, p)
}

func (s *Server) handleGetProject(c *gin.Context) {
	id := c.Param("id")
	if !s.requireMembership(c, id) {
		return
	}
	p, err := s.store.GetProject(id)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "项目不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载项目失败")
		return
	}
	OK(c, p)
}
