package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/issue"
	"github.com/ningpm/ningpm/apps/api/internal/domain/notification"
)

func (s *Server) handleListNotifications(c *gin.Context) {
	list, err := s.store.ListNotifications(currentUserID(c), 40)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载通知失败")
		return
	}
	if list == nil {
		list = []notification.Notification{}
	}
	OK(c, list)
}

func (s *Server) handleUnreadNotificationCount(c *gin.Context) {
	n, err := s.store.CountUnreadNotifications(currentUserID(c))
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载未读数失败")
		return
	}
	OK(c, gin.H{"count": n})
}

func (s *Server) handleMarkNotificationRead(c *gin.Context) {
	if err := s.store.MarkNotificationRead(c.Param("id"), currentUserID(c)); err != nil {
		Fail(c, http.StatusInternalServerError, "标记已读失败")
		return
	}
	OK(c, gin.H{"ok": true})
}

func (s *Server) handleMarkAllNotificationsRead(c *gin.Context) {
	if err := s.store.MarkAllNotificationsRead(currentUserID(c)); err != nil {
		Fail(c, http.StatusInternalServerError, "标记已读失败")
		return
	}
	OK(c, gin.H{"ok": true})
}

// notifyAssigned alerts the new assignee; skip when assigning to self.
func (s *Server) notifyAssigned(actorID string, it *issue.Issue) {
	if it == nil || it.AssigneeID == "" || it.AssigneeID == actorID {
		return
	}
	_ = s.store.CreateNotification(it.AssigneeID, actorID, notification.KindAssigned, it, "")
}

// notifyCommented alerts issue creator and assignee (excluding the commenter).
func (s *Server) notifyCommented(actorID string, it *issue.Issue, body string) {
	if it == nil {
		return
	}
	seen := map[string]struct{}{}
	for _, uid := range []string{it.AssigneeID, it.CreatorID} {
		if uid == "" || uid == actorID {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		_ = s.store.CreateNotification(uid, actorID, notification.KindCommented, it, body)
	}
}
