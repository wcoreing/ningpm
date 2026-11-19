package user

import (
	"strings"
	"time"
)

// User 是系统账号。Account 为登录标识；Name 为显示用户名；Email 可选联系邮箱。
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Account      string    `json:"account"`
	Email        string    `json:"email,omitempty"`
	Avatar       string    `json:"-"` // resource id；空则前端用默认字母头像
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"isAdmin"`
	CreatedAt    time.Time `json:"createdAt"`
}

// AvatarURL returns public avatar path when a custom file exists.
func (u *User) AvatarURL() string {
	if u == nil || strings.TrimSpace(u.Avatar) == "" {
		return ""
	}
	return "/api/avatars/" + u.ID
}
