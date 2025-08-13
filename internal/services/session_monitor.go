package services

import (
	"log"
	"sync"
	"time"

	"very-jump/internal/database/models"
)

// SessionMonitor 会话监控服务
type SessionMonitor struct {
	sessionService *models.SessionService
	ttydService    *TTYDService
	stopChan       chan struct{}
	wg             sync.WaitGroup
	isRunning      bool
	mutex          sync.Mutex
	
	// 配置参数
	checkInterval   time.Duration // 检查间隔
	sessionTimeout  time.Duration // 会话超时时间
}

// NewSessionMonitor 创建会话监控服务
func NewSessionMonitor(sessionService *models.SessionService, ttydService *TTYDService) *SessionMonitor {
	return &SessionMonitor{
		sessionService: sessionService,
		ttydService:    ttydService,
		stopChan:       make(chan struct{}),
		checkInterval:  5 * time.Minute,  // 每5分钟检查一次
		sessionTimeout: 30 * time.Minute, // 30分钟无心跳视为超时
	}
}

// SetConfig 设置监控配置
func (sm *SessionMonitor) SetConfig(checkInterval, sessionTimeout time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.checkInterval = checkInterval
	sm.sessionTimeout = sessionTimeout
}

// Start 启动会话监控
func (sm *SessionMonitor) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	if sm.isRunning {
		return nil
	}
	
	sm.isRunning = true
	sm.wg.Add(1)
	
	go sm.monitorLoop()
	
	log.Printf("Session monitor started - check interval: %v, session timeout: %v", 
		sm.checkInterval, sm.sessionTimeout)
	
	return nil
}

// Stop 停止会话监控
func (sm *SessionMonitor) Stop() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	if !sm.isRunning {
		return
	}
	
	sm.isRunning = false
	close(sm.stopChan)
	sm.wg.Wait()
	
	log.Printf("Session monitor stopped")
}

// monitorLoop 监控循环
func (sm *SessionMonitor) monitorLoop() {
	defer sm.wg.Done()
	
	ticker := time.NewTicker(sm.checkInterval)
	defer ticker.Stop()
	
	// 启动时立即执行一次清理
	sm.cleanupStaleSessions()
	
	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.cleanupStaleSessions()
		}
	}
}

// cleanupStaleSessions 清理超时会话
func (sm *SessionMonitor) cleanupStaleSessions() {
	// 获取超时的活跃会话
	staleSessions, err := sm.sessionService.GetStaleActiveSessions(sm.sessionTimeout)
	if err != nil {
		log.Printf("Failed to get stale sessions: %v", err)
		return
	}
	
	if len(staleSessions) == 0 {
		log.Printf("No stale sessions found")
		return
	}
	
	log.Printf("Found %d stale sessions to clean up", len(staleSessions))
	
	// 清理数据库中的会话状态
	cleanedCount, err := sm.sessionService.CleanupStaleActiveSessions(sm.sessionTimeout)
	if err != nil {
		log.Printf("Failed to cleanup stale sessions in database: %v", err)
		return
	}
	
	// 清理对应的TTYD进程
	var cleanedProcessCount int
	if sm.ttydService != nil {
		for _, session := range staleSessions {
			// 查找对应的TTYD进程
			for _, process := range sm.ttydService.ListActiveSessions() {
				if process.DBSessionID == session.ID {
					if err := sm.ttydService.StopTTYDSession(process.SessionID); err != nil {
						log.Printf("Failed to stop TTYD session %s: %v", process.SessionID, err)
					} else {
						cleanedProcessCount++
						log.Printf("Stopped stale TTYD session: %s (DB session: %s)", 
							process.SessionID, session.ID)
					}
					break
				}
			}
		}
	}
	
	log.Printf("Session cleanup completed - DB sessions: %d, TTYD processes: %d", 
		cleanedCount, cleanedProcessCount)
}

// GetStatus 获取监控状态
func (sm *SessionMonitor) GetStatus() map[string]interface{} {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	return map[string]interface{}{
		"running":          sm.isRunning,
		"check_interval":   sm.checkInterval.String(),
		"session_timeout":  sm.sessionTimeout.String(),
	}
}

// ForceCleanup 强制执行一次清理
func (sm *SessionMonitor) ForceCleanup() {
	go sm.cleanupStaleSessions()
}