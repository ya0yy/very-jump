package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"very-jump/internal/database/models"
	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
)

// TerminalHandler 终端处理器
type TerminalHandler struct {
	ttydService   *services.TTYDService
	serverService *models.ServerService
}

// NewTerminalHandler 创建终端处理器
func NewTerminalHandler(ttydService *services.TTYDService, serverService *models.ServerService) *TerminalHandler {
	return &TerminalHandler{
		ttydService:   ttydService,
		serverService: serverService,
	}
}

// StartTerminalResponse 启动终端响应
type StartTerminalResponse struct {
	SessionID string `json:"session_id"`
	Port      int    `json:"port"`
	URL       string `json:"url"`
}

// StartTerminal 启动终端会话
func (h *TerminalHandler) StartTerminal(c *gin.Context) {
	serverID, err := strconv.Atoi(c.Param("server_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	server, err := h.serverService.GetByID(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	if role != "admin" {
		hasPermission, err := h.checkUserServerPermission(userID.(int), serverID, c)
		if err != nil || !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有访问该服务器的权限"})
			return
		}
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	process, err := h.ttydService.StartTTYDSessionWithAudit(server, userID.(int), username.(string), ipAddress, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("启动终端失败: %v", err)})
		return
	}

	url := fmt.Sprintf("/proxy-terminal/?session_id=%s", process.SessionID)

	response := StartTerminalResponse{
		SessionID: process.SessionID,
		Port:      process.Port,
		URL:       url,
	}

	c.JSON(http.StatusOK, response)
}

// ProxyToTTYD 代理请求到ttyd
func (h *TerminalHandler) ProxyToTTYD(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.String(http.StatusBadRequest, "Query parameter 'session_id' is required")
		return
	}

	process, exists := h.ttydService.GetTTYDProcess(sessionID)
	if !exists {
		c.String(http.StatusNotFound, "Session not found")
		return
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", process.Port))
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating proxy target")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// This is the critical part for WebSocket proxying.
	// We need to hijack the connection and handle the upgrade manually.
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = c.Request.URL.Path
		req.Host = target.Host
	}

	// For regular HTTP requests, use the standard proxy.
	proxy.ServeHTTP(c.Writer, c.Request)
}

// StopTerminal 停止终端会话
func (h *TerminalHandler) StopTerminal(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	process, exists := h.ttydService.GetTTYDProcess(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if role != "admin" && process.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限停止该会话"})
		return
	}

	if err := h.ttydService.StopTTYDSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("停止终端失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "终端会话已停止"})
}

// GetTerminalInfo 获取终端信息
func (h *TerminalHandler) GetTerminalInfo(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	process, exists := h.ttydService.GetTTYDProcess(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if role != "admin" && process.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问该会话"})
		return
	}

	url := fmt.Sprintf("/proxy-terminal/?session_id=%s", process.SessionID)

	response := gin.H{
		"session_id": process.SessionID,
		"port":       process.Port,
		"url":        url,
		"server_id":  process.ServerID,
		"username":   process.Username,
		"created_at": process.CreatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// ListActiveSessions 列出活跃会话
func (h *TerminalHandler) ListActiveSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	sessions := h.ttydService.ListActiveSessions()

	if role != "admin" {
		filteredSessions := make([]*services.TTYDProcess, 0)
		for _, session := range sessions {
			if session.UserID == userID.(int) {
				filteredSessions = append(filteredSessions, session)
			}
		}
		sessions = filteredSessions
	}

	response := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, gin.H{
			"session_id": session.SessionID,
			"port":       session.Port,
			"server_id":  session.ServerID,
			"username":   session.Username,
			"created_at": session.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}

// checkUserServerPermission 检查用户服务器权限
func (h *TerminalHandler) checkUserServerPermission(userID, serverID int, c *gin.Context) (bool, error) {
	// 这里应该调用数据库检查权限，暂时简化实现
	_ = userID
	_ = serverID
	_ = c
	return true, nil
}
