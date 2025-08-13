package api

import (
	"log"
	"net/http"

	"very-jump/internal/database/models"
	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	serverService *models.ServerService
	userService   *models.UserService
	auditService  *services.AuditService
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(ss *models.ServerService, us *models.UserService, as *services.AuditService) *StatsHandler {
	return &StatsHandler{
		serverService: ss,
		userService:   us,
		auditService:  as,
	}
}

// GetStats 获取系统统计信息
func (h *StatsHandler) GetStats(c *gin.Context) {
	log.Printf("Getting server count...")
	serverCount, err := h.serverService.GetServerCount()
	if err != nil {
		log.Printf("Error getting server count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get server count"})
		return
	}
	log.Printf("Server count: %d", serverCount)

	log.Printf("Getting user count...")
	userCount, err := h.userService.GetUserCount()
	if err != nil {
		log.Printf("Error getting user count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user count"})
		return
	}
	log.Printf("User count: %d", userCount)

	log.Printf("Getting audit statistics...")
	auditStats, err := h.auditService.GetAuditStatistics(c)
	if err != nil {
		log.Printf("Error getting audit statistics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit statistics"})
		return
	}
	log.Printf("Audit statistics: %+v", auditStats)

	c.JSON(http.StatusOK, gin.H{
		"total_servers":   serverCount,
		"active_sessions": auditStats.ActiveSessions,
		"total_users":     userCount,
		"total_sessions":  auditStats.TotalSessions,
	})
}
