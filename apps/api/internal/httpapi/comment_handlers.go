package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/domain/comment"
)

type createCommentReq struct {
	Body string `json:"body" binding:"required"`
}

func (s *Server) handleListComments(c *gin.Context) {
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
	list, err := s.store.ListComments(issueID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载评论失败")
		return
	}
	if list == nil {
		list = []comment.Comment{}
	}
	OK(c, list)
}

func (s *Server) handleCreateComment(c *gin.Context) {
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
	if !s.requireWrite(c, it.ProjectID) {
		return
	}
	var req createCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "请填写评论内容")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		Fail(c, http.StatusBadRequest, "评论不能为空")
		return
	}
	actorID := currentUserID(c)
	cm, err := s.store.CreateComment(issueID, actorID, body)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "发表评论失败")
		return
	}
	s.notifyCommented(actorID, it, body)
	OK(c, cm)
}
