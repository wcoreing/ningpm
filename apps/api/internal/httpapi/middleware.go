package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ctxUserID = "userID"
const ctxUserName = "userName"
const ctxUserAccount = "userAccount"

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		claims, err := s.jwt.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			Fail(c, http.StatusUnauthorized, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxUserName, claims.Name)
		c.Set(ctxUserAccount, claims.Account)
		c.Next()
	}
}

func currentUserID(c *gin.Context) string {
	v, _ := c.Get(ctxUserID)
	s, _ := v.(string)
	return s
}
