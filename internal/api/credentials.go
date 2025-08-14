package api

import (
	"net/http"
	"strconv"

	"very-jump/internal/database/models"

	"github.com/gin-gonic/gin"
)

// CredentialHandler 登录凭证处理器
type CredentialHandler struct {
	credentialService *models.CredentialService
}

// NewCredentialHandler 创建登录凭证处理器
func NewCredentialHandler(credentialService *models.CredentialService) *CredentialHandler {
	return &CredentialHandler{
		credentialService: credentialService,
	}
}

// List 获取登录凭证列表
func (h *CredentialHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	credentials, err := h.credentialService.List(pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.credentialService.GetCredentialCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credentials": credentials,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// Get 获取单个登录凭证
func (h *CredentialHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的凭证ID"})
		return
	}

	credential, err := h.credentialService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "凭证不存在"})
		return
	}

	c.JSON(http.StatusOK, credential)
}

// Create 创建登录凭证
func (h *CredentialHandler) Create(c *gin.Context) {
	var req models.CredentialCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := h.credentialService.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, credential)
}

// Update 更新登录凭证
func (h *CredentialHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的凭证ID"})
		return
	}

	var req models.CredentialUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credential, err := h.credentialService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, credential)
}

// Delete 删除登录凭证
func (h *CredentialHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的凭证ID"})
		return
	}

	err = h.credentialService.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "凭证已删除"})
}
