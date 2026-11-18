package httpapi

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/cycle"
)

type createCycleReq struct {
	Name      string       `json:"name" binding:"required"`
	Goal      string       `json:"goal"`
	Status    cycle.Status `json:"status"`
	StartDate string       `json:"startDate" binding:"required"`
	EndDate   string       `json:"endDate" binding:"required"`
}

type updateCycleReq struct {
	Name      *string       `json:"name"`
	Goal      *string       `json:"goal"`
	Status    *cycle.Status `json:"status"`
	StartDate *string       `json:"startDate"`
	EndDate   *string       `json:"endDate"`
}

func (s *Server) handleListCycles(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireMembership(c, projectID) {
		return
	}
	list, err := s.store.ListCycles(projectID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载迭代失败")
		return
	}
	if list == nil {
		list = []cycle.Cycle{}
	}
	OK(c, list)
}

func (s *Server) handleCreateCycle(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireWrite(c, projectID) {
		return
	}
	var req createCycleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写迭代名称与日期")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "迭代名称不能为空")
		return
	}
	if !validDate(req.StartDate) || !validDate(req.EndDate) {
		Fail(c, http.StatusBadRequest, "日期格式应为 YYYY-MM-DD")
		return
	}
	if req.EndDate < req.StartDate {
		Fail(c, http.StatusBadRequest, "结束日期不能早于开始日期")
		return
	}
	st := req.Status
	if st == "" {
		st = cycle.StatusPlanned
	}
	if !validCycleStatus(st) {
		Fail(c, http.StatusBadRequest, "迭代状态无效")
		return
	}
	cy := &cycle.Cycle{
		ProjectID: projectID,
		Name:      name,
		Goal:      strings.TrimSpace(req.Goal),
		Status:    st,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}
	if err := s.store.CreateCycle(cy); err != nil {
		Fail(c, http.StatusInternalServerError, "创建迭代失败")
		return
	}
	OK(c, cy)
}

func (s *Server) handleUpdateCycle(c *gin.Context) {
	projectID := c.Param("id")
	cycleID := c.Param("cycleId")
	if !s.requireWrite(c, projectID) {
		return
	}
	cy, err := s.store.GetCycle(cycleID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "迭代不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载迭代失败")
		return
	}
	if cy.ProjectID != projectID {
		Fail(c, http.StatusNotFound, "迭代不存在")
		return
	}
	var req updateCycleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请求无效")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			Fail(c, http.StatusBadRequest, "迭代名称不能为空")
			return
		}
		cy.Name = name
	}
	if req.Goal != nil {
		cy.Goal = strings.TrimSpace(*req.Goal)
	}
	if req.Status != nil {
		if !validCycleStatus(*req.Status) {
			Fail(c, http.StatusBadRequest, "迭代状态无效")
			return
		}
		cy.Status = *req.Status
	}
	if req.StartDate != nil {
		if !validDate(*req.StartDate) {
			Fail(c, http.StatusBadRequest, "开始日期格式无效")
			return
		}
		cy.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		if !validDate(*req.EndDate) {
			Fail(c, http.StatusBadRequest, "结束日期格式无效")
			return
		}
		cy.EndDate = *req.EndDate
	}
	if cy.EndDate < cy.StartDate {
		Fail(c, http.StatusBadRequest, "结束日期不能早于开始日期")
		return
	}
	if err := s.store.UpdateCycle(cy); err != nil {
		Fail(c, http.StatusInternalServerError, "更新迭代失败")
		return
	}
	OK(c, cy)
}

func validCycleStatus(st cycle.Status) bool {
	switch st {
	case cycle.StatusPlanned, cycle.StatusActive, cycle.StatusClosed:
		return true
	default:
		return false
	}
}

func validDate(v string) bool {
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}
