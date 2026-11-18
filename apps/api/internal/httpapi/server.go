package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/auth"
	"github.com/ningpm/ningpm/apps/api/internal/config"
	"github.com/ningpm/ningpm/apps/api/internal/store"
)

// Server wires HTTP handlers to dependencies.
type Server struct {
	cfg   config.Config
	store *store.Store
	jwt   *auth.Manager
}

// NewServer creates an API server.
func NewServer(cfg config.Config, st *store.Store, jwt *auth.Manager) *Server {
	return &Server{cfg: cfg, store: st, jwt: jwt}
}

// NewRouter builds the Gin engine and mounts routes.
func (s *Server) NewRouter() *gin.Engine {
	if s.cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(corsDev())

	api := r.Group("/api")
	{
		api.GET("/health", handleHealth)
		api.GET("/version", handleVersion)

		api.POST("/auth/login", s.handleLogin)
		api.GET("/meta/labels", s.handleLabels)

		authz := api.Group("")
		authz.Use(s.requireAuth())
		{
			authz.GET("/auth/me", s.handleMe)
			authz.PATCH("/auth/me", s.handleUpdateMe)
			authz.POST("/auth/password", s.handleChangePassword)
			authz.POST("/auth/avatar", s.handleUploadAvatar)
			authz.DELETE("/auth/avatar", s.handleDeleteAvatar)
			authz.GET("/avatars/:userId", s.handleGetAvatar)
			authz.GET("/users", s.handleListUsers)

			authz.GET("/projects", s.handleListProjects)
			authz.POST("/projects", s.handleCreateProject)
			authz.GET("/projects/:id", s.handleGetProject)
			authz.PATCH("/projects/:id", s.handleUpdateProject)
			authz.GET("/projects/:id/members", s.handleListMembers)
			authz.POST("/projects/:id/members", s.handleAddMember)
			authz.PATCH("/projects/:id/members/:userId", s.handleUpdateMember)
			authz.DELETE("/projects/:id/members/:userId", s.handleRemoveMember)
			authz.GET("/projects/:id/cycles", s.handleListCycles)
			authz.POST("/projects/:id/cycles", s.handleCreateCycle)
			authz.PATCH("/projects/:id/cycles/:cycleId", s.handleUpdateCycle)
			authz.GET("/projects/:id/modules", s.handleListModules)
			authz.POST("/projects/:id/modules", s.handleCreateModule)
			authz.DELETE("/projects/:id/modules/:moduleId", s.handleDeleteModule)
			authz.GET("/projects/:id/attachments", s.handleListProjectAttachments)
			authz.GET("/projects/:id/files", s.handleListProjectFiles)
			authz.POST("/projects/:id/files", s.handleAddProjectFile)
			authz.DELETE("/projects/:id/files/:fileId", s.handleRemoveProjectFile)

			authz.GET("/issues", s.handleListIssues)
			authz.POST("/issues", s.handleCreateIssue)
			authz.GET("/issues/:id", s.handleGetIssue)
			authz.PATCH("/issues/:id", s.handleUpdateIssue)
			authz.PATCH("/issues/:id/status", s.handleUpdateIssueStatus)
			authz.GET("/issues/:id/comments", s.handleListComments)
			authz.POST("/issues/:id/comments", s.handleCreateComment)
			authz.GET("/issues/:id/attachments", s.handleListIssueAttachments)
			authz.POST("/issues/:id/attachments", s.handleAddIssueAttachment)
			authz.DELETE("/issues/:id/attachments/:attachmentId", s.handleRemoveIssueAttachment)

			authz.POST("/resources", s.handleCreateResource)
			authz.GET("/resources/:id", s.handleGetResource)
			authz.DELETE("/resources/:id", s.handleDeleteResource)

			authz.GET("/notifications", s.handleListNotifications)
			authz.GET("/notifications/unread-count", s.handleUnreadNotificationCount)
			authz.POST("/notifications/read-all", s.handleMarkAllNotificationsRead)
			authz.POST("/notifications/:id/read", s.handleMarkNotificationRead)
		}
	}

	return r
}

func corsDev() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
