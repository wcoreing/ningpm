package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issue"
	"github.com/ningpm/ningpm/apps/api/internal/store"
)

type createIssueReq struct {
	ProjectID   string         `json:"projectId" binding:"required"`
	Type        issue.Type     `json:"type" binding:"required"`
	Title       string         `json:"title" binding:"required"`
	Description string         `json:"description"`
	Priority    issue.Priority `json:"priority"`
	AssigneeID  string         `json:"assigneeId"`
	CycleID     string         `json:"cycleId"`
	ModuleID    string         `json:"moduleId"`
	ParentID    string         `json:"parentId"`
	Status      issue.Status   `json:"status"`
}

type updateIssueReq struct {
	Type        *issue.Type     `json:"type"`
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	Priority    *issue.Priority `json:"priority"`
	AssigneeID  *string         `json:"assigneeId"`
	CycleID     *string         `json:"cycleId"`
	ModuleID    *string         `json:"moduleId"`
	Status      *issue.Status   `json:"status"`
}

type updateStatusReq struct {
	Status issue.Status `json:"status" binding:"required"`
}

func (s *Server) handleListIssues(c *gin.Context) {
	rootsQ := strings.ToLower(strings.TrimSpace(c.Query("rootsOnly")))
	includeChildren := c.Query("includeChildren") == "1" || c.Query("includeChildren") == "true" ||
		rootsQ == "0" || rootsQ == "false"
	f := store.IssueFilter{
		ProjectID:  c.Query("projectId"),
		Type:       c.Query("type"),
		Status:     c.Query("status"),
		AssigneeID: c.Query("assigneeId"),
		CycleID:    c.Query("cycleId"),
		ModuleID:   c.Query("moduleId"),
		ParentID:   c.Query("parentId"),
		RootsOnly:  rootsQ == "1" || rootsQ == "true",
		Backlog:    c.Query("backlog") == "1" || c.Query("backlog") == "true",
		UserID:     currentUserID(c),
		MineOnly:   c.Query("mine") == "1" || c.Query("mine") == "true",
	}
	// 看板默认只看根事项；includeChildren / rootsOnly=0 / 显式 parentId / 我的待办则不过滤。
	if includeChildren {
		f.RootsOnly = false
	} else if f.ParentID == "" && !f.MineOnly && rootsQ == "" {
		f.RootsOnly = true
	}
	if f.ProjectID != "" && !s.requireMembership(c, f.ProjectID) {
		return
	}
	list, err := s.store.ListIssues(f)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载事项失败")
		return
	}
	if list == nil {
		list = []issue.Issue{}
	}
	OK(c, list)
}

func (s *Server) handleCreateIssue(c *gin.Context) {
	var req createIssueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写事项类型和标题")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		Fail(c, http.StatusBadRequest, "标题不能为空")
		return
	}
	if !validType(req.Type) {
		Fail(c, http.StatusBadRequest, "事项类型无效")
		return
	}
	if _, err := s.store.GetProject(req.ProjectID); err != nil {
		Fail(c, http.StatusBadRequest, "所属项目不存在")
		return
	}
	if !s.requireWrite(c, req.ProjectID) {
		return
	}
	assigneeID := strings.TrimSpace(req.AssigneeID)
	if assigneeID != "" {
		if _, err := s.store.GetMember(req.ProjectID, assigneeID); err != nil {
			Fail(c, http.StatusBadRequest, "指派人必须是项目成员")
			return
		}
	}
	cycleID := strings.TrimSpace(req.CycleID)
	if cycleID != "" {
		cy, err := s.store.GetCycle(cycleID)
		if err != nil || cy.ProjectID != req.ProjectID {
			Fail(c, http.StatusBadRequest, "所属迭代无效")
			return
		}
	}
	moduleID := strings.TrimSpace(req.ModuleID)
	if moduleID != "" {
		md, err := s.store.GetModule(moduleID)
		if err != nil || md.ProjectID != req.ProjectID {
			Fail(c, http.StatusBadRequest, "所属模块无效")
			return
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID != "" {
		parent, err := s.store.GetIssue(parentID)
		if err != nil {
			Fail(c, http.StatusBadRequest, "父事项不存在")
			return
		}
		if parent.ProjectID != req.ProjectID {
			Fail(c, http.StatusBadRequest, "父事项不属于该项目")
			return
		}
		if parent.ParentID != "" {
			Fail(c, http.StatusBadRequest, "子事项不能再拆子事项")
			return
		}
		if cycleID == "" {
			cycleID = parent.CycleID
		}
		if moduleID == "" {
			moduleID = parent.ModuleID
		}
	}
	priority := req.Priority
	if priority == 0 {
		priority = issue.PriorityMedium
	}
	status := req.Status
	if status == "" {
		status = issue.StatusTodo
	}
	if !validStatus(status) {
		Fail(c, http.StatusBadRequest, "状态无效")
		return
	}
	it := &issue.Issue{
		ProjectID:   req.ProjectID,
		Type:        req.Type,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		Priority:    priority,
		AssigneeID:  assigneeID,
		CycleID:     cycleID,
		ModuleID:    moduleID,
		ParentID:    parentID,
		CreatorID:   currentUserID(c),
	}
	if err := s.store.CreateIssue(it); err != nil {
		Fail(c, http.StatusInternalServerError, "创建事项失败")
		return
	}
	s.notifyAssigned(currentUserID(c), it)
	OK(c, it)
}

func (s *Server) handleGetIssue(c *gin.Context) {
	it, err := s.store.GetIssue(c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "事项不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载事项失败")
		return
	}
	if !s.requireMembership(c, it.ProjectID) {
		return
	}
	OK(c, it)
}

func (s *Server) handleUpdateIssue(c *gin.Context) {
	it, err := s.store.GetIssue(c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "事项不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载事项失败")
		return
	}
	if !s.requireWrite(c, it.ProjectID) {
		return
	}
	prevAssignee := it.AssigneeID
	var req updateIssueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请求无效")
		return
	}
	if req.Type != nil {
		if !validType(*req.Type) {
			Fail(c, http.StatusBadRequest, "事项类型无效")
			return
		}
		it.Type = *req.Type
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			Fail(c, http.StatusBadRequest, "标题不能为空")
			return
		}
		it.Title = title
	}
	if req.Description != nil {
		it.Description = strings.TrimSpace(*req.Description)
	}
	if req.Priority != nil {
		it.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		assigneeID := strings.TrimSpace(*req.AssigneeID)
		if assigneeID != "" {
			if _, err := s.store.GetMember(it.ProjectID, assigneeID); err != nil {
				Fail(c, http.StatusBadRequest, "指派人必须是项目成员")
				return
			}
		}
		it.AssigneeID = assigneeID
	}
	if req.CycleID != nil {
		cycleID := strings.TrimSpace(*req.CycleID)
		if cycleID != "" {
			cy, err := s.store.GetCycle(cycleID)
			if err != nil || cy.ProjectID != it.ProjectID {
				Fail(c, http.StatusBadRequest, "所属迭代无效")
				return
			}
		}
		it.CycleID = cycleID
	}
	if req.ModuleID != nil {
		moduleID := strings.TrimSpace(*req.ModuleID)
		if moduleID != "" {
			md, err := s.store.GetModule(moduleID)
			if err != nil || md.ProjectID != it.ProjectID {
				Fail(c, http.StatusBadRequest, "所属模块无效")
				return
			}
		}
		it.ModuleID = moduleID
	}
	if req.Status != nil {
		if !validStatus(*req.Status) {
			Fail(c, http.StatusBadRequest, "状态无效")
			return
		}
		it.Status = *req.Status
	}
	if err := s.store.UpdateIssue(it); err != nil {
		Fail(c, http.StatusInternalServerError, "更新事项失败")
		return
	}
	if req.AssigneeID != nil && it.AssigneeID != "" && it.AssigneeID != prevAssignee {
		s.notifyAssigned(currentUserID(c), it)
	}
	OK(c, it)
}

func (s *Server) handleUpdateIssueStatus(c *gin.Context) {
	var req updateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil || !validStatus(req.Status) {
		Fail(c, http.StatusBadRequest, "状态无效")
		return
	}
	current, err := s.store.GetIssue(c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "事项不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载事项失败")
		return
	}
	if !s.requireWrite(c, current.ProjectID) {
		return
	}
	it, err := s.store.UpdateIssueStatus(c.Param("id"), req.Status)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "更新状态失败")
		return
	}
	OK(c, it)
}

func validType(t issue.Type) bool {
	switch t {
	case issue.TypeStory, issue.TypeTask, issue.TypeBug, issue.TypeEpic:
		return true
	default:
		return false
	}
}

func validStatus(st issue.Status) bool {
	switch st {
	case issue.StatusTodo, issue.StatusDoing, issue.StatusReview, issue.StatusDone:
		return true
	default:
		return false
	}
}
