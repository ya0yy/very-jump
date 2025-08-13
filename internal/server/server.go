package server

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"

	"very-jump/internal/api"
	"very-jump/internal/config"
	"very-jump/internal/database/models"
	"very-jump/internal/middleware"
	"very-jump/internal/services"

	"github.com/gin-gonic/gin"
)

// Server HTTP 服务器
type Server struct {
	cfg          *config.Config
	db           *sql.DB
	router       *gin.Engine
	ttydService  *services.TTYDService
	auditService *services.AuditService
}

// New 创建服务器
func New(cfg *config.Config, db *sql.DB) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 初始化审计服务
	auditService := services.NewAuditService(db)

	// 初始化会话服务
	sessionService := models.NewSessionService(db)

	// 初始化ttyd服务
	ttydService := services.NewTTYDService(cfg.DataDir, auditService, sessionService)

	return &Server{
		cfg:          cfg,
		db:           db,
		router:       router,
		ttydService:  ttydService,
		auditService: auditService,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.setupMiddleware()
	s.setupRoutes()

	log.Printf("Server starting on port %s", s.cfg.Port)
	return s.router.Run(":" + s.cfg.Port)
}

// setupMiddleware 设置中间件
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.LoggingMiddleware())
	s.router.Use(middleware.CORSMiddleware())
	s.router.Use(gin.Recovery())
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 静态文件服务（前端构建文件）
	s.router.Static("/assets", "./web/dist/assets")
	s.router.StaticFile("/vite.svg", "./web/dist/vite.svg")
	s.router.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

	// 创建服务
	authService := services.NewAuthService(s.cfg, s.db)
	serverService := models.NewServerService(s.db)
	userService := models.NewUserService(s.db)
	sessionService := models.NewSessionService(s.db)
	// auditLogService := models.NewAuditLogService(s.db)

	// 创建处理器
	authHandler := api.NewAuthHandler(authService)
	serverHandler := api.NewServerHandler(serverService)
	userHandler := api.NewUserHandler(userService)
	recordingsDir := filepath.Join(s.cfg.DataDir, "recordings")
	sessionHandler := api.NewSessionHandler(sessionService, recordingsDir, s.ttydService)
	statsHandler := api.NewStatsHandler(serverService, userService, s.auditService)
	// auditLogHandler := api.NewAuditLogHandler(auditLogService)
	terminalHandler := api.NewTerminalHandler(s.ttydService, serverService)
	auditHandler := api.NewAuditHandler(s.auditService, s.ttydService)

	// API 路由
	apiV1 := s.router.Group("/api/v1")
	{
		// 认证路由
		auth := apiV1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/profile", middleware.AuthMiddleware(s.cfg), authHandler.Profile)
		}

		// 需要认证的路由
		authenticated := apiV1.Group("")
		authenticated.Use(middleware.AuthMiddleware(s.cfg))
		{
			// 服务器管理
			servers := authenticated.Group("/servers")
			{
				servers.GET("", serverHandler.List)
				servers.GET("/:id", serverHandler.Get)
			}

			// 会话管理
			sessions := authenticated.Group("/sessions")
			{
				sessions.GET("", sessionHandler.List)
				sessions.GET("/:id", sessionHandler.Get)
				sessions.POST("/:id/close", sessionHandler.Close)
				sessions.GET("/active", sessionHandler.GetActiveSessions)
			}

			// 审计日志 (旧版，保留兼容性)
			// auditLogs := authenticated.Group("/audit-logs")
			// {
			//     auditLogs.GET("", auditLogHandler.List)
			// }

			// 管理员路由
			admin := authenticated.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				// 服务器管理（管理员）
				admin.POST("/servers", serverHandler.Create)
				admin.PUT("/servers/:id", serverHandler.Update)
				admin.DELETE("/servers/:id", serverHandler.Delete)

				// 用户管理
				admin.GET("/users", userHandler.List)
				admin.POST("/users", userHandler.Create)
				admin.GET("/users/:id", userHandler.Get)
				admin.PUT("/users/:id", userHandler.Update)
				admin.DELETE("/users/:id", userHandler.Delete)

				// 系统统计
				admin.GET("/stats", statsHandler.GetStats)
			}
		}

		// 终端路由 (ttyd)
		terminal := authenticated.Group("/terminal")
		{
			terminal.POST("/start/:server_id", terminalHandler.StartTerminal)
			terminal.POST("/stop/:session_id", terminalHandler.StopTerminal)
			terminal.GET("/info/:session_id", terminalHandler.GetTerminalInfo)
			terminal.GET("/sessions", terminalHandler.ListActiveSessions)
		}

		// 审计管理路由
		audit := authenticated.Group("/audit")
		{
			audit.GET("/logs", auditHandler.GetAuditLogs)
			audit.GET("/sessions", auditHandler.GetTerminalSessions)
			audit.GET("/statistics", auditHandler.GetAuditStatistics)
			audit.GET("/alerts", auditHandler.GetSecurityAlerts)
			audit.PUT("/alerts/:alert_id/resolve", auditHandler.ResolveSecurityAlert)
		}

	}

	// 健康检查
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// 首页路由
	s.router.GET("/", func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	// 终端代理路由（使用query参数避免路径冲突）
	s.router.Any("/proxy-terminal/*proxy-path", middleware.AuthMiddleware(s.cfg), terminalHandler.ProxyToTTYD)

	// SPA 路由处理（前端路由）
	s.router.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}
