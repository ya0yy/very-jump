package services

import (
	"database/sql"
	"errors"
	"time"

	"very-jump/internal/config"
	"very-jump/internal/database/models"
	"very-jump/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService 认证服务
type AuthService struct {
	cfg         *config.Config
	userService *models.UserService
}

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, db *sql.DB) *AuthService {
	return &AuthService{
		cfg:         cfg,
		userService: models.NewUserService(db),
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	user, err := s.userService.GetByUsername(req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	if !user.ValidatePassword(req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 生成 JWT Token
	expiresAt := time.Now().Add(s.cfg.JWTExpiry)
	claims := &middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "very-jump",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     tokenString,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// GetUserService 获取用户服务
func (s *AuthService) GetUserService() *models.UserService {
	return s.userService
}
