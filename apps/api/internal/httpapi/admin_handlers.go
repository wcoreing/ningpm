package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := s.store.GetUserByID(currentUserID(c))
		if err != nil || u == nil || !u.IsAdmin {
			Fail(c, http.StatusForbidden, "需要系统管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

type adminCreateUserReq struct {
	Name     string `json:"name" binding:"required"` // 用户名（显示名）
	Account  string `json:"account"`                 // 登录账号，选填；不填则按姓名生成
	Email    string `json:"email"`                   // legacy alias for account
	Password string `json:"password" binding:"required"`
	IsAdmin  bool   `json:"isAdmin"`
}

type adminUpdateUserReq struct {
	Name     string `json:"name"`
	Account  string `json:"account"`
	Email    string `json:"email"` // legacy alias for account
	Password string `json:"password"`
	IsAdmin  *bool  `json:"isAdmin"`
}

func (s *Server) handleAdminListUsers(c *gin.Context) {
	users, err := s.store.ListUsers()
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载账号失败")
		return
	}
	out := make([]gin.H, 0, len(users))
	for i := range users {
		out = append(out, userAdminPublic(&users[i]))
	}
	OK(c, out)
}

func (s *Server) handleAdminCreateUser(c *gin.Context) {
	var req adminCreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写用户名和密码")
		return
	}
	name := strings.TrimSpace(req.Name)
	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Email)
	}
	pwd := strings.TrimSpace(req.Password)
	if name == "" {
		Fail(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if len(pwd) < 6 {
		Fail(c, http.StatusBadRequest, "密码至少 6 位")
		return
	}
	account, err := s.store.AllocateLoginAccount(account, name)
	if err != nil {
		if err.Error() == "account taken" {
			Fail(c, http.StatusConflict, "该账号已存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "创建失败")
		return
	}
	u, err := s.store.CreateUserAdmin(name, account, pwd, req.IsAdmin)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "创建失败")
		return
	}
	OK(c, userAdminPublic(u))
}

func (s *Server) handleAdminUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	var req adminUpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数无效")
		return
	}
	pwd := strings.TrimSpace(req.Password)
	if pwd != "" && len(pwd) < 6 {
		Fail(c, http.StatusBadRequest, "密码至少 6 位")
		return
	}

	existing, err := s.store.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "账号不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "更新失败")
		return
	}

	if req.IsAdmin != nil && !*req.IsAdmin && existing.IsAdmin {
		n, err := s.store.CountAdmins()
		if err != nil {
			Fail(c, http.StatusInternalServerError, "更新失败")
			return
		}
		if n <= 1 {
			Fail(c, http.StatusBadRequest, "至少保留一名系统管理员")
			return
		}
	}

	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Email)
	}
	u, err := s.store.UpdateUserAdmin(userID, req.Name, account, pwd, req.IsAdmin)
	if err != nil {
		if err.Error() == "email taken" || err.Error() == "account taken" {
			Fail(c, http.StatusConflict, "该账号已被占用")
			return
		}
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "账号不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "更新失败")
		return
	}
	OK(c, userAdminPublic(u))
}

func (s *Server) handleAdminDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == currentUserID(c) {
		Fail(c, http.StatusBadRequest, "不能删除当前登录账号")
		return
	}
	target, err := s.store.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "账号不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "删除失败")
		return
	}
	if target.IsAdmin {
		n, err := s.store.CountAdmins()
		if err != nil {
			Fail(c, http.StatusInternalServerError, "删除失败")
			return
		}
		if n <= 1 {
			Fail(c, http.StatusBadRequest, "至少保留一名系统管理员")
			return
		}
	}
	if err := s.store.DeleteUser(userID, currentUserID(c)); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "账号不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "删除失败")
		return
	}
	OK(c, gin.H{"deleted": true})
}
