package user

import (
	"strings"
	"time"
)

// User 是系统账号。
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Avatar       string    `json:"-"` // resource id；空则前端用默认字母头像
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// AvatarURL returns public avatar path when a custom file exists.
func (u *User) AvatarURL() string {
	if u == nil || strings.TrimSpace(u.Avatar) == "" {
		return ""
	}
	return "/api/avatars/" + u.ID
}
