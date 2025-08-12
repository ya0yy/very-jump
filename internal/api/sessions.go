package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"very-jump/internal/database/models"

	"github.com/gin-gonic/gin"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	sessionService *models.SessionService
	recordingsDir  string               // 录制文件目录
	ttydService    TTYDServiceInterface // 添加TTYD服务接口
}

// TTYDServiceInterface TTYD服务接口
type TTYDServiceInterface interface {
	GetRecordingsInfo() (int, int64, error)
	CleanupRecordings(maxAge time.Duration)
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(sessionService *models.SessionService, recordingsDir string, ttydService TTYDServiceInterface) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
		recordingsDir:  recordingsDir,
		ttydService:    ttydService,
	}
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

// Replay 获取会话录制文件进行回放
func (h *SessionHandler) Replay(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	// 获取会话信息
	session, err := h.sessionService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 非管理员只能回放自己的会话
	if role != "admin" && session.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限回放该会话"})
		return
	}

	// 检查录制文件是否存在
	if session.RecordingFile == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "该会话没有录制文件"})
		return
	}

	recordingPath := filepath.Join(h.recordingsDir, session.RecordingFile)

	// 检查文件是否存在
	if _, err := os.Stat(recordingPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "录制文件不存在"})
		return
	}

	// 打开录制文件
	file, err := os.Open(recordingPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开录制文件"})
		return
	}
	defer file.Close()

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "inline; filename="+session.RecordingFile)

	// 流式传输文件内容
	if _, err := io.Copy(c.Writer, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "传输录制文件失败"})
		return
	}
}

// GetReplayInfo 获取会话回放信息
func (h *SessionHandler) GetReplayInfo(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	// 获取会话信息
	session, err := h.sessionService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 非管理员只能查看自己的会话回放信息
	if role != "admin" && session.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限查看该会话"})
		return
	}

	// 检查录制文件是否存在
	hasRecording := false
	recordingSize := int64(0)

	if session.RecordingFile != "" {
		recordingPath := filepath.Join(h.recordingsDir, session.RecordingFile)
		if stat, err := os.Stat(recordingPath); err == nil {
			hasRecording = true
			recordingSize = stat.Size()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session":        session,
		"has_recording":  hasRecording,
		"recording_size": recordingSize,
	})
}

// GetRecordingsStats 获取录制文件统计信息 (管理员)
func (h *SessionHandler) GetRecordingsStats(c *gin.Context) {
	role, _ := c.Get("role")

	// 只有管理员可以查看录制文件统计
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问录制文件统计"})
		return
	}

	if h.ttydService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTYD服务不可用"})
		return
	}

	fileCount, totalSize, err := h.ttydService.GetRecordingsInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取录制文件统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_count":    fileCount,
		"total_size":    totalSize,
		"total_size_mb": float64(totalSize) / (1024 * 1024),
	})
}

// CleanupOldRecordings 清理旧录制文件 (管理员)
func (h *SessionHandler) CleanupOldRecordings(c *gin.Context) {
	role, _ := c.Get("role")

	// 只有管理员可以清理录制文件
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限清理录制文件"})
		return
	}

	if h.ttydService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTYD服务不可用"})
		return
	}

	// 默认清理30天前的录制文件
	maxAge := 30 * 24 * time.Hour
	if days := c.Query("days"); days != "" {
		if parsed, err := strconv.Atoi(days); err == nil && parsed > 0 {
			maxAge = time.Duration(parsed) * 24 * time.Hour
		}
	}

	h.ttydService.CleanupRecordings(maxAge)

	c.JSON(http.StatusOK, gin.H{
		"message":      "录制文件清理完成",
		"max_age_days": int(maxAge.Hours() / 24),
	})
}
