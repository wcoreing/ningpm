package member

import "time"

// Role is a project membership role.
type Role string

const (
	RoleOwner  Role = "owner"  // 负责人
	RoleAdmin  Role = "admin"  // 管理员
	RoleMember Role = "member" // 成员
	RoleViewer Role = "viewer" // 只读
)

// Member is a user belonging to a project.
type Member struct {
	ProjectID string    `json:"projectId"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
	Role      Role      `json:"role"`
	JoinedAt  time.Time `json:"joinedAt"`
}

// Label returns Chinese role name.
func Label(r Role) string {
	switch r {
	case RoleOwner:
		return "负责人"
	case RoleAdmin:
		return "管理员"
	case RoleMember:
		return "成员"
	case RoleViewer:
		return "只读"
	default:
		return string(r)
	}
}

// CanManage returns whether the role can manage members.
func CanManage(r Role) bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanWrite returns whether the role can create/edit issues.
func CanWrite(r Role) bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}
