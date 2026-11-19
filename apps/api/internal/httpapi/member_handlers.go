package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/member"
)

type addMemberReq struct {
	UserID  string      `json:"userId"`
	Account string      `json:"account"`
	Email   string      `json:"email"` // legacy alias for account
	Name    string      `json:"name"`
	Role    member.Role `json:"role"`
}

type addMemberResp struct {
	Member              member.Member `json:"member"`
	CreatedUser         bool          `json:"createdUser"`
	TemporaryPassword   string        `json:"temporaryPassword,omitempty"`
}

type updateMemberReq struct {
	Role member.Role `json:"role" binding:"required"`
}

func (s *Server) handleListMembers(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireMembership(c, projectID) {
		return
	}
	list, err := s.store.ListMembers(projectID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载成员失败")
		return
	}
	if list == nil {
		list = []member.Member{}
	}
	OK(c, list)
}

func (s *Server) handleAddMember(c *gin.Context) {
	projectID := c.Param("id")
	me, ok := s.requireManage(c, projectID)
	if !ok {
		return
	}
	_ = me

	var req addMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写账号")
		return
	}
	role := req.Role
	if role == "" {
		role = member.RoleMember
	}
	if !validRole(role) || role == member.RoleOwner {
		Fail(c, http.StatusBadRequest, "角色无效（负责人不可直接添加，请先加为管理员再转让）")
		return
	}

	userID := strings.TrimSpace(req.UserID)
	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Email)
	}
	createdUser := false
	tempPassword := ""

	if userID == "" {
		if account == "" {
			Fail(c, http.StatusBadRequest, "请填写登录账号")
			return
		}
		u, temp, created, err := s.store.EnsureUserByAccount(account, strings.TrimSpace(req.Name))
		if err != nil {
			Fail(c, http.StatusInternalServerError, "创建账号失败")
			return
		}
		userID = u.ID
		createdUser = created
		tempPassword = temp
	} else if _, err := s.store.GetUserByID(userID); err != nil {
		Fail(c, http.StatusBadRequest, "用户不存在")
		return
	}

	if _, err := s.store.GetMember(projectID, userID); err == nil {
		Fail(c, http.StatusConflict, "该用户已在项目中")
		return
	} else if err != sql.ErrNoRows {
		Fail(c, http.StatusInternalServerError, "添加成员失败")
		return
	}

	m, err := s.store.AddMember(projectID, userID, role)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "添加成员失败")
		return
	}
	OK(c, addMemberResp{
		Member:            *m,
		CreatedUser:       createdUser,
		TemporaryPassword: tempPassword,
	})
}

func (s *Server) handleUpdateMember(c *gin.Context) {
	projectID := c.Param("id")
	targetID := c.Param("userId")
	me, ok := s.requireManage(c, projectID)
	if !ok {
		return
	}

	var req updateMemberReq
	if err := c.ShouldBindJSON(&req); err != nil || !validRole(req.Role) {
		Fail(c, http.StatusBadRequest, "角色无效")
		return
	}

	target, err := s.store.GetMember(projectID, targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "成员不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载成员失败")
		return
	}

	if target.Role == member.RoleOwner && req.Role != member.RoleOwner {
		n, err := s.store.CountOwners(projectID)
		if err != nil {
			Fail(c, http.StatusInternalServerError, "更新角色失败")
			return
		}
		if n <= 1 {
			Fail(c, http.StatusBadRequest, "不能取消唯一负责人")
			return
		}
	}
	if req.Role == member.RoleOwner && me.Role != member.RoleOwner {
		Fail(c, http.StatusForbidden, "仅负责人可转让负责人角色")
		return
	}

	m, err := s.store.UpdateMemberRole(projectID, targetID, req.Role)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "更新角色失败")
		return
	}
	OK(c, m)
}

func (s *Server) handleRemoveMember(c *gin.Context) {
	projectID := c.Param("id")
	targetID := c.Param("userId")
	uid := currentUserID(c)
	selfLeave := targetID == uid

	var me *member.Member
	if selfLeave {
		if !s.requireMembership(c, projectID) {
			return
		}
		m, err := s.store.GetMember(projectID, uid)
		if err != nil {
			Fail(c, http.StatusInternalServerError, "校验成员失败")
			return
		}
		me = m
	} else {
		var ok bool
		me, ok = s.requireManage(c, projectID)
		if !ok {
			return
		}
	}

	target, err := s.store.GetMember(projectID, targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "成员不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载成员失败")
		return
	}
	if target.Role == member.RoleOwner {
		n, _ := s.store.CountOwners(projectID)
		if n <= 1 {
			Fail(c, http.StatusBadRequest, "不能移除唯一负责人")
			return
		}
	}
	if selfLeave && me.Role == member.RoleOwner {
		n, _ := s.store.CountOwners(projectID)
		if n <= 1 {
			Fail(c, http.StatusBadRequest, "负责人不能退出（请先转让）")
			return
		}
	}

	if err := s.store.RemoveMember(projectID, targetID); err != nil {
		Fail(c, http.StatusInternalServerError, "移除成员失败")
		return
	}
	OK(c, gin.H{"removed": true})
}

func (s *Server) requireMembership(c *gin.Context, projectID string) bool {
	if _, err := s.store.GetProject(projectID); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "项目不存在")
		} else {
			Fail(c, http.StatusInternalServerError, "加载项目失败")
		}
		return false
	}
	if _, err := s.store.GetMember(projectID, currentUserID(c)); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusForbidden, "你不是该项目成员")
			return false
		}
		Fail(c, http.StatusInternalServerError, "校验成员失败")
		return false
	}
	return true
}

func (s *Server) requireManage(c *gin.Context, projectID string) (*member.Member, bool) {
	if _, err := s.store.GetProject(projectID); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "项目不存在")
		} else {
			Fail(c, http.StatusInternalServerError, "加载项目失败")
		}
		return nil, false
	}
	m, err := s.store.GetMember(projectID, currentUserID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusForbidden, "你不是该项目成员")
			return nil, false
		}
		Fail(c, http.StatusInternalServerError, "校验成员失败")
		return nil, false
	}
	if !member.CanManage(m.Role) {
		Fail(c, http.StatusForbidden, "需要管理员或负责人权限")
		return nil, false
	}
	return m, true
}

func (s *Server) requireWrite(c *gin.Context, projectID string) bool {
	m, err := s.store.GetMember(projectID, currentUserID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusForbidden, "你不是该项目成员")
			return false
		}
		Fail(c, http.StatusInternalServerError, "校验成员失败")
		return false
	}
	if !member.CanWrite(m.Role) {
		Fail(c, http.StatusForbidden, "只读成员不能修改事项")
		return false
	}
	return true
}

func validRole(r member.Role) bool {
	switch r {
	case member.RoleOwner, member.RoleAdmin, member.RoleMember, member.RoleViewer:
		return true
	default:
		return false
	}
}

