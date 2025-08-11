package api

import (
	"net/http"
	"strconv"

	"very-jump/internal/database/models"

	"github.com/gin-gonic/gin"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	sessionService *models.SessionService
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(sessionService *models.SessionService) *SessionHandler {
	return &SessionHandler{sessionService: sessionService}
}

// List 获取会话列表
func (h *SessionHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	var sessions []*models.Session
	var err error

	// 管理员可以看到所有会话，普通用户只能看到自己的会话
	if role == "admin" {
		sessions, err = h.sessionService.List(limit, offset)
	} else {
		sessions, err = h.sessionService.GetByUserID(userID.(int), limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// Get 获取会话详情
func (h *SessionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	session, err := h.sessionService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 非管理员只能查看自己的会话
	if role != "admin" && session.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限查看该会话"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// Close 关闭会话
func (h *SessionHandler) Close(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	session, err := h.sessionService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 非管理员只能关闭自己的会话
	if role != "admin" && session.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限关闭该会话"})
		return
	}

	if err := h.sessionService.Close(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "会话已关闭"})
}

// GetActiveSessions 获取活跃会话统计
func (h *SessionHandler) GetActiveSessions(c *gin.Context) {
	count, err := h.sessionService.GetActiveSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active_sessions": count,
	})
}
