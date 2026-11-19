package httpapi

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issue"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issueevent"
)

func (s *Server) handleListIssueEvents(c *gin.Context) {
	issueID := c.Param("id")
	it, err := s.store.GetIssue(issueID)
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
	list, err := s.store.ListIssueEvents(issueID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载操作日志失败")
		return
	}
	OK(c, list)
}

func (s *Server) handleListProjectEvents(c *gin.Context) {
	projectID := c.Param("id")
	if _, err := s.store.GetProject(projectID); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "项目不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载项目失败")
		return
	}
	if !s.requireMembership(c, projectID) {
		return
	}
	list, err := s.store.ListProjectEvents(projectID, 40)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载操作日志失败")
		return
	}
	OK(c, list)
}

func (s *Server) recordIssueEvent(id, issueID, actorID, kind, summary string, at time.Time) {
	summary = strings.TrimSpace(summary)
	if issueID == "" || summary == "" {
		return
	}
	_ = s.store.AddIssueEvent(issueevent.Event{
		ID: id, IssueID: issueID, ActorID: actorID, Kind: kind, Summary: summary, CreatedAt: at,
	})
}

func (s *Server) recordIssueDiff(actorID string, before, after *issue.Issue) {
	if before == nil || after == nil {
		return
	}
	var parts []string
	if before.Title != after.Title {
		parts = append(parts, "把标题改为「"+clipRunes(after.Title, 40)+"」")
	}
	if before.Type != after.Type {
		parts = append(parts, "把类型从"+issue.TypeLabel(before.Type)+"改为"+issue.TypeLabel(after.Type))
	}
	if before.Status != after.Status {
		parts = append(parts, "把状态从"+issue.StatusLabel(before.Status)+"改为"+issue.StatusLabel(after.Status))
	}
	if before.Priority != after.Priority {
		parts = append(parts, "把优先级从"+issue.PriorityLabel(before.Priority)+"改为"+issue.PriorityLabel(after.Priority))
	}
	if before.AssigneeID != after.AssigneeID {
		parts = append(parts, "把指派从"+blankAs(before.AssigneeName, "未指派")+"改为"+blankAs(after.AssigneeName, "未指派"))
	}
	if before.CycleID != after.CycleID {
		parts = append(parts, "把迭代从"+blankAs(before.CycleName, "待规划")+"改为"+blankAs(after.CycleName, "待规划"))
	}
	if before.ModuleID != after.ModuleID {
		parts = append(parts, "把模块从"+blankAs(before.ModuleName, "无")+"改为"+blankAs(after.ModuleName, "无"))
	}
	if before.StartDate != after.StartDate {
		parts = append(parts, "把开始日从"+blankAs(before.StartDate, "未设置")+"改为"+blankAs(after.StartDate, "未设置"))
	}
	if before.DueDate != after.DueDate {
		parts = append(parts, "把截止日期从"+blankAs(before.DueDate, "未设置")+"改为"+blankAs(after.DueDate, "未设置"))
	}
	if before.Description != after.Description {
		parts = append(parts, "修改了描述")
	}
	if len(parts) == 0 {
		return
	}
	s.recordIssueEvent("", after.ID, actorID, "update", strings.Join(parts, "；"), time.Now().UTC())
}

func blankAs(v, empty string) string {
	if strings.TrimSpace(v) == "" {
		return empty
	}
	return v
}

func clipRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
