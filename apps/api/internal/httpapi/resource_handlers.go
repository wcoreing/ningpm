package httpapi

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxResourceBytes = 20 << 20 // 20MB

var allowedResourceExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
	".pdf": true, ".txt": true, ".md": true, ".csv": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".zip": true, ".json": true, ".log": true,
}

func (s *Server) handleCreateResource(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, "请选择文件")
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

	ct := file.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	res, err := s.store.CreateResourceFromReader(currentUserID(c), file.Filename, ct, file.Size, src)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "保存失败")
		return
	}
	OK(c, res)
}

func (s *Server) handleGetResource(c *gin.Context) {
	id := c.Param("id")
	res, err := s.store.GetResource(id)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "资源不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载失败")
		return
	}
	uid := currentUserID(c)
	ok, projectIDs, err := s.store.ResourceAccessInfo(id)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "校验权限失败")
		return
	}
	allowed := res.UploaderID == uid
	if !allowed && ok.AsAvatar {
		allowed = true
	}
	if !allowed && len(projectIDs) > 0 {
		for _, pid := range projectIDs {
			if m, _ := s.store.GetMember(pid, uid); m != nil {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		Fail(c, http.StatusForbidden, "无权访问该资源")
		return
	}
	path := s.store.ResourceAbsPath(res.StorageKey)
	if path == "" {
		Fail(c, http.StatusNotFound, "文件不存在")
		return
	}
	if _, err := os.Stat(path); err != nil {
		Fail(c, http.StatusNotFound, "文件不存在")
		return
	}
	c.Header("Content-Type", res.ContentType)
	c.Header("Content-Disposition", `inline; filename="`+res.OriginalName+`"`)
	c.File(path)
}

func (s *Server) handleDeleteResource(c *gin.Context) {
	id := c.Param("id")
	res, err := s.store.GetResource(id)
	if err != nil {
		if err == sql.ErrNoRows {
			Fail(c, http.StatusNotFound, "资源不存在")
			return
		}
		Fail(c, http.StatusInternalServerError, "加载失败")
		return
	}
	if res.UploaderID != currentUserID(c) {
		Fail(c, http.StatusForbidden, "只能删除自己上传的资源")
		return
	}
	if err := s.store.DeleteResource(id); err != nil {
		if err.Error() == "resource still referenced" {
			Fail(c, http.StatusConflict, "资源仍被引用，无法删除")
			return
		}
		Fail(c, http.StatusInternalServerError, "删除失败")
		return
	}
	OK(c, gin.H{"deleted": true, "id": id})
}
