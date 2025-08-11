package main

import (
	"log"
	"os"
	"path/filepath"
	"very-jump/internal/database"

	"very-jump/internal/config"
	"very-jump/internal/server"
)

func main() {
	// 确保数据目录存在
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// 创建必要的目录
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "sessions"),
		filepath.Join(dataDir, "config"),
		filepath.Join(dataDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// 加载配置
	cfg := config.Load(dataDir)

	// 初始化数据库
	db, err := database.Init(filepath.Join(dataDir, "very-jump.db"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 启动服务器
	srv := server.New(cfg, db)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
