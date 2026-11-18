package httpapi

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type attachResourceReq struct {
	ResourceID string `json:"resourceId"`
}

func (s *Server) handleListProjectAttachments(c *gin.Context) {
	projectID := c.Param("id")
	if !s.requireMembership(c, projectID) {
		return
	}
	list, err := s.store.ListProjectAttachments(projectID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载附件失败")
		return
	}
	OK(c, list)
}

func (s *Server) handleListIssueAttachments(c *gin.Context) {
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
	list, err := s.store.ListIssueAttachments(issueID)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "加载附件失败")
		return
	}
	OK(c, list)
}

func (s *Server) handleAddIssueAttachment(c *gin.Context) {
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

	ct := c.ContentType()
	var resourceID string
	if strings.HasPrefix(ct, "multipart/") {
		file, err := c.FormFile("file")
		if err != nil {
			Fail(c, http.StatusBadRequest, "请选择文件或传入 resourceId")
			return
		}
		if file.Size > maxResourceBytes {
			Fail(c, http.StatusBadRequest, "文件不能超过 20MB")
			return
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedResourceExt[ext] {
			Fail(c, http.StatusBadRequest, "不支持的文件类型")
			return
		}
		src, err := file.Open()
		if err != nil {
			Fail(c, http.StatusInternalServerError, "读取文件失败")
			return
		}
		defer src.Close()
		fileCT := file.Header.Get("Content-Type")
		if fileCT == "" {
			fileCT = "application/octet-stream"
		}
		res, err := s.store.CreateResourceFromReader(currentUserID(c), file.Filename, fileCT, file.Size, src)
		if err != nil {
			Fail(c, http.StatusInternalServerError, "保存失败")
			return
		}
		resourceID = res.ID
	} else {
		var req attachResourceReq
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ResourceID) == "" {
			Fail(c, http.StatusBadRequest, "请提供 resourceId 或上传文件")
			return
		}
		resourceID = strings.TrimSpace(req.ResourceID)
	}

	a, err := s.store.AttachResourceToIssue(issueID, resourceID, currentUserID(c))
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "资源不存在")
			return
		}
		Fail(c, http.StatusConflict, "挂接附件失败（可能已存在）")
		return
	}
	OK(c, a)
}

func (s *Server) handleRemoveIssueAttachment(c *gin.Context) {
	issueID := c.Param("id")
	attachmentID := c.Param("attachmentId")
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
	if err := s.store.DetachIssueAttachment(issueID, attachmentID); err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "附件不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "删除附件失败")
		return
	}
	OK(c, gin.H{"removed": true})
}
