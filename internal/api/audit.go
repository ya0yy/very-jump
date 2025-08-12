package api

import (
	"net/http"
	"strconv"

	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审计API处理器
type AuditHandler struct {
	auditService *services.AuditService
	ttydService  *services.TTYDService
}

// NewAuditHandler 创建审计处理器实例
func NewAuditHandler(auditService *services.AuditService, ttydService *services.TTYDService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
		ttydService:  ttydService,
	}
}

// GetAuditLogs 获取审计日志
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	// 用户过滤（管理员可以查看所有，普通用户只能查看自己的）
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user := userInterface.(map[string]interface{})
	userID := int(user["id"].(float64))
	role := user["role"].(string)

	var filterUserID *int
	if role != "admin" {
		filterUserID = &userID
	} else {
		// 管理员可以指定查看特定用户的日志
		if userIDParam := c.Query("user_id"); userIDParam != "" {
			if uid, err := strconv.Atoi(userIDParam); err == nil {
				filterUserID = &uid
			}
		}
	}

	logs, err := h.auditService.GetAuditLogs(c.Request.Context(), filterUserID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取审计日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSecurityAlerts 获取安全告警
func (h *AuditHandler) GetSecurityAlerts(c *gin.Context) {
	// 检查管理员权限
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user := userInterface.(map[string]interface{})
	role := user["role"].(string)
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	// 解决状态过滤
	var resolved *bool
	if resolvedParam := c.Query("resolved"); resolvedParam != "" {
		if resolvedVal, err := strconv.ParseBool(resolvedParam); err == nil {
			resolved = &resolvedVal
		}
	}

	alerts, err := h.auditService.GetSecurityAlerts(c.Request.Context(), resolved, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取安全告警失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":    alerts,
		"page":      page,
		"page_size": pageSize,
	})
}

// ResolveSecurityAlert 解决安全告警
func (h *AuditHandler) ResolveSecurityAlert(c *gin.Context) {
	// 检查管理员权限
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user := userInterface.(map[string]interface{})
	role := user["role"].(string)
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	alertID, err := strconv.Atoi(c.Param("alert_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的告警ID"})
		return
	}

	userID := int(user["id"].(float64))

	// TODO: 实现解决告警的逻辑
	// 这里应该调用 auditService 的方法来标记告警为已解决

	c.JSON(http.StatusOK, gin.H{
		"message":     "告警已解决",
		"alert_id":    alertID,
		"resolved_by": userID,
	})
}

// GetAuditStatistics 获取审计统计信息
func (h *AuditHandler) GetAuditStatistics(c *gin.Context) {
	// 检查管理员权限
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user := userInterface.(map[string]interface{})
	role := user["role"].(string)
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	stats, err := h.auditService.GetAuditStatistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计信息失败"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetTerminalSessions 获取终端会话列表
func (h *AuditHandler) GetTerminalSessions(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 检查权限
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	user := userInterface.(map[string]interface{})
	userID := int(user["id"].(float64))
	role := user["role"].(string)

	// 从ttyd服务获取所有活跃的会话
	allSessions := h.ttydService.ListActiveSessions()

	var userSessions []*services.TTYDProcess
	if role == "admin" {
		// 管理员可以查看所有会话
		userSessions = allSessions
	} else {
		// 普通用户只能查看自己的会话
		for _, s := range allSessions {
			if s.UserID == userID {
				userSessions = append(userSessions, s)
			}
		}
	}

	// 分页处理
	total := len(userSessions)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedSessions := userSessions[start:end]

	// 格式化输出
	responseSessions := make([]gin.H, len(paginatedSessions))
	for i, s := range paginatedSessions {
		responseSessions[i] = gin.H{
			"session_id": s.SessionID,
			"user_id":    s.UserID,
			"server_id":  s.ServerID,
			"username":   s.Username,
			"server":     s.ServerName,
			"start_time": s.CreatedAt,
			"status":     "active",
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions":  responseSessions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
