package httpapi

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Account  string `json:"account"`
	Email    string `json:"email"` // legacy alias for account
	Password string `json:"password" binding:"required"`
}

type updateMeReq struct {
	Name string `json:"name" binding:"required"`
}

type changePasswordReq struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

func userPublic(u *user.User) gin.H {
	account := u.Account
	if account == "" {
		account = u.Email
	}
	out := gin.H{
		"id":      u.ID,
		"name":    u.Name,
		"account": account,
		"isAdmin": u.IsAdmin,
	}
	// email 仅作可选联系字段；与 account 相同时不对外重复暴露
	if u.Email != "" && u.Email != account {
		out["email"] = u.Email
	}
	if url := u.AvatarURL(); url != "" {
		out["avatarUrl"] = url
	}
	return out
}

func userAdminPublic(u *user.User) gin.H {
	out := userPublic(u)
	out["createdAt"] = u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	return out
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写账号和密码")
		return
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Email)
	}
	if account == "" {
		Fail(c, http.StatusBadRequest, "请填写账号和密码")
		return
	}
	u, err := s.store.GetUserByAccount(account)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusUnauthorized, "账号或密码错误")
			return
		}
		Fail(c, http.StatusInternalServerError, "登录失败")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		Fail(c, http.StatusUnauthorized, "账号或密码错误")
		return
	}
	loginID := u.Account
	if loginID == "" {
		loginID = u.Email
	}
	token, err := s.jwt.Issue(u.ID, u.Name, loginID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "签发令牌失败")
		return
	}
	OK(c, gin.H{
		"token": token,
		"user":  userPublic(u),
	})
}

func (s *Server) handleMe(c *gin.Context) {
	u, err := s.store.GetUserByID(currentUserID(c))
	if err != nil {
		Fail(c, http.StatusUnauthorized, "用户不存在")
		return
	}
	OK(c, userPublic(u))
}

func (s *Server) handleUpdateMe(c *gin.Context) {
	var req updateMeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写显示名")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, "显示名不能为空")
		return
	}
	u, err := s.store.UpdateUserName(currentUserID(c), name)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusUnauthorized, "用户不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "更新失败")
		return
	}
	OK(c, userPublic(u))
}

func (s *Server) handleChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写当前密码和新密码")
		return
	}
	newPwd := strings.TrimSpace(req.NewPassword)
	if len(newPwd) < 6 {
		Fail(c, http.StatusBadRequest, "新密码至少 6 位")
		return
	}
	err := s.store.UpdateUserPassword(currentUserID(c), req.CurrentPassword, newPwd)
	if err != nil {
		if err.Error() == "current password mismatch" {
			Fail(c, http.StatusBadRequest, "当前密码不正确")
			return
		}
		if err == sql.ErrNoRows {
			Fail(c, http.StatusUnauthorized, "用户不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "修改密码失败")
		return
	}
	OK(c, gin.H{"changed": true})
}

func (s *Server) handleUploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择图片文件")
		return
	}
	if file.Size > 2<<20 {
		Fail(c, http.StatusBadRequest, "头像不能超过 2MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		Fail(c, http.StatusBadRequest, "仅支持 jpg / png / webp / gif")
		return
	}
	src, err := file.Open()
	if err != nil {
		Fail(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	defer src.Close()
	ct := file.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	res, err := s.store.CreateResourceFromReader(currentUserID(c), file.Filename, ct, file.Size, src)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "保存失败")
		return
	}
	u, err := s.store.SetUserAvatarResource(currentUserID(c), res.ID)
	if err != nil {
		_ = s.store.ForceDeleteResource(res.ID)
		Fail(c, http.StatusInternalServerError, "更新头像失败")
		return
	}
	OK(c, userPublic(u))
}

func (s *Server) handleDeleteAvatar(c *gin.Context) {
	u, err := s.store.ClearUserAvatarResource(currentUserID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusUnauthorized, "用户不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "清除头像失败")
		return
	}
	OK(c, userPublic(u))
}

func (s *Server) handleGetAvatar(c *gin.Context) {
	id := c.Param("userId")
	u, err := s.store.GetUserByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "用户不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载失败")
		return
	}
	if u.Avatar == "" {
		Fail(c, http.StatusNotFound, "未设置头像")
		return
	}
	res, err := s.store.GetResource(u.Avatar)
	if err != nil {
		Fail(c, http.StatusNotFound, "头像文件不存在")
		return
	}
	path := s.store.ResourceAbsPath(res.StorageKey)
	if path == "" {
		Fail(c, http.StatusNotFound, "头像文件不存在")
		return
	}
	if _, err := os.Stat(path); err != nil {
		Fail(c, http.StatusNotFound, "头像文件不存在")
		return
	}
	c.Header("Content-Type", res.ContentType)
	c.File(path)
}

func (s *Server) handleListUsers(c *gin.Context) {
	users, err := s.store.ListUsers()
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载用户失败")
		return
	}
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		account := u.Account
		if account == "" {
			account = u.Email
		}
		item := gin.H{"id": u.ID, "name": u.Name, "account": account}
		if u.Email != "" {
			item["email"] = u.Email
		}
		if url := u.AvatarURL(); url != "" {
			item["avatarUrl"] = url
		}
		out = append(out, item)
	}
	OK(c, out)
}

func (s *Server) handleLabels(c *gin.Context) {
	OK(c, gin.H{
		"types": []gin.H{
			{"value": "story", "label": "需求"},
			{"value": "task", "label": "任务"},
			{"value": "bug", "label": "缺陷"},
			{"value": "epic", "label": "史诗"},
		},
		"statuses": []gin.H{
			{"value": "todo", "label": "待处理"},
			{"value": "doing", "label": "进行中"},
			{"value": "review", "label": "待验收"},
			{"value": "done", "label": "已完成"},
		},
		"priorities": []gin.H{
			{"value": 1, "label": "紧急"},
			{"value": 2, "label": "高"},
			{"value": 3, "label": "中"},
			{"value": 4, "label": "低"},
		},
		"roles": []gin.H{
			{"value": "owner", "label": "负责人"},
			{"value": "admin", "label": "管理员"},
			{"value": "member", "label": "成员"},
			{"value": "viewer", "label": "只读"},
		},
		"moduleStatuses": []gin.H{
			{"value": "planned", "label": "规划中"},
			{"value": "active", "label": "进行中"},
			{"value": "completed", "label": "已完成"},
		},
	})
}
