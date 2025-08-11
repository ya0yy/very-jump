package api

import (
	"net/http"

	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Profile 获取用户信息
func (h *AuthHandler) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	user, err := h.authService.GetUserService().GetByID(userID.(int))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT 是无状态的，这里只返回成功响应
	// 实际的 token 失效由客户端处理
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}
