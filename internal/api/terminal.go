package api

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"very-jump/internal/database/models"
	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TerminalHandler 终端处理器
type TerminalHandler struct {
	ttydService   *services.TTYDService
	serverService *models.ServerService
	upgrader      websocket.Upgrader
}

// NewTerminalHandler 创建终端处理器
func NewTerminalHandler(ttydService *services.TTYDService, serverService *models.ServerService) *TerminalHandler {
	return &TerminalHandler{
		ttydService:   ttydService,
		serverService: serverService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该更严格
			},
		},
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

	// 更新服务器上次登录时间
	if err := h.serverService.UpdateLastLoginTime(serverID); err != nil {
		log.Printf("Failed to update last login time for server %d: %v", serverID, err)
		// 不影响主流程，只记录错误
	}

	url := fmt.Sprintf("/proxy-terminal/?session_id=%s", process.SessionID)

	response := StartTerminalResponse{
		SessionID: process.SessionID,
		Port:      process.Port,
		URL:       url,
	}

	c.JSON(http.StatusOK, response)
}

// ProxyToTTYD 代理请求到ttyd，支持录制
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

	// 构造后端目标 URL
	targetURL := fmt.Sprintf("http://127.0.0.1:%d", process.Port)

	if c.IsWebsocket() {
		log.Printf("Handling WebSocket request for session %s", process.SessionID)
		h.handleWebSocketWithRecording(c, process)
		return
	} else {
		log.Printf("Handling HTTP request for session %s", process.SessionID)
	}

	// 处理普通 HTTP 请求 - 标准代理（原有逻辑）
	target, err := url.Parse(targetURL)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating proxy target")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = c.Request.URL.Path
		req.Host = target.Host
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// handleWebSocketWithRecording 处理WebSocket连接并录制数据
func (h *TerminalHandler) handleWebSocketWithRecording(c *gin.Context, process *services.TTYDProcess) {
	log.Printf("WebSocket upgrade attempt for session %s, path: %s, headers: %v",
		process.SessionID, c.Request.URL.Path, c.Request.Header)

	rheader := http.Header{
		"sec-websocket-protocol": []string{"tty"},
	}
	// 升级客户端连接为WebSocket
	clientConn, err := h.upgrader.Upgrade(c.Writer, c.Request, rheader)
	if err != nil {
		log.Printf("Failed to upgrade client connection for session %s: %v", process.SessionID, err)
		return
	}
	defer func() {
		log.Printf("Closing client connection for session %s", process.SessionID)
		clientConn.Close()
	}()

	ttydPath := "/proxy-terminal/ws"
	ttydURL := fmt.Sprintf("ws://127.0.0.1:%d%s", process.Port, ttydPath)

	log.Printf("Connecting to ttyd WebSocket: %s", ttydURL)

	// 连接到ttyd
	ttydConn, _, err := websocket.DefaultDialer.Dial(ttydURL, rheader)
	if err != nil {
		log.Printf("Failed to connect to ttyd WebSocket: %v", err)
		clientConn.WriteMessage(websocket.CloseMessage, []byte("Failed to connect to terminal"))
		return
	}
	defer ttydConn.Close()

	log.Printf("WebSocket proxy with recording established for session %s", process.SessionID)

	// 使用channel来同步两个goroutine
	clientDone := make(chan struct{})
	ttydDone := make(chan struct{})

	// 客户端 -> ttyd (用户输入)
	go func() {
		defer close(clientDone)
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Client WebSocket read error: %v", err)
				}
				break
			}

			// 录制用户输入（只录制文本消息）

			if messageType == websocket.BinaryMessage && process.Recorder != nil && process.Recorder.IsRecording() {
				if err := process.Recorder.WriteInput(message); err != nil {
					log.Printf("Failed to record input: %v", err)
				}
			}

			// 转发到ttyd
			if err := ttydConn.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to forward to ttyd: %v", err)
				break
			}
		}
	}()

	// ttyd -> 客户端 (终端输出)
	go func() {
		defer close(ttydDone)
		for {
			messageType, message, err := ttydConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("TTYD WebSocket read error: %v", err)
				}
				break
			}

			// 录制终端输出（只录制文本消息）
			if messageType == websocket.BinaryMessage && process.Recorder != nil && process.Recorder.IsRecording() {
				if err := process.Recorder.WriteOutput(message); err != nil {
					log.Printf("Failed to record output: %v", err)
				}
			}

			// 转发到客户端
			if err := clientConn.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to forward to client: %v", err)
				break
			}
		}
	}()

	// 等待任一方向的连接关闭
	select {
	case <-clientDone:
		log.Printf("Client connection closed for session %s", process.SessionID)
	case <-ttydDone:
		log.Printf("TTYD connection closed for session %s", process.SessionID)
	}

	log.Printf("WebSocket proxy with recording closed for session %s", process.SessionID)
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
