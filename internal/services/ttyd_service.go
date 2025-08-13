package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"very-jump/internal/database/models"
)

// SSH连接的expect脚本内容
const sshConnectExpScript = `#!/usr/bin/expect -f

# 获取命令行参数
set username [lindex $argv 0]
set password [lindex $argv 1]
set host [lindex $argv 2]
set port [lindex $argv 3]

# 设置超时时间
set timeout 30

# 关闭命令回显
log_user 0

# 启动 SSH 连接
spawn ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null $username@$host -p $port

# 等待密码提示并输入密码
expect {
    "*password:*" {
        send "$password\r"
        # 重新开启命令回显
        log_user 1
        expect {
            "*$*" {
                # 成功登录，进入交互模式
                interact
            }
            "*#*" {
                # 成功登录（root用户），进入交互模式
                interact
            }
            timeout {
                puts "Login failed"
                exit 1
            }
        }
    }
    timeout {
        puts "Connection timeout"
        exit 1
    }
    eof {
        puts "Connection failed"
        exit 1
    }
}
`

// TTYDService ttyd服务
type TTYDService struct {
	dataDir        string
	processes      map[string]*TTYDProcess // key: sessionID, value: ttyd进程信息
	mutex          sync.RWMutex
	basePort       int           // 基础端口号，从7681开始
	auditService   *AuditService // 审计服务
	sessionService *models.SessionService
	recordingsDir  string // 录制文件存储目录
}

// TTYDProcess ttyd进程信息
type TTYDProcess struct {
	SessionID     string
	UserID        int
	Username      string
	ServerID      int
	ServerName    string // 添加服务器名称字段
	Port          int
	Process       *os.Process
	Cancel        context.CancelFunc
	CreatedAt     time.Time
	RecordingFile string           // 录制文件路径
	DBSessionID   string           // 数据库中的会话ID
	Recorder      *SessionRecorder // 录制器
}

// NewTTYDService 创建ttyd服务
func NewTTYDService(dataDir string, auditService *AuditService, sessionService *models.SessionService) *TTYDService {
	recordingsDir := filepath.Join(dataDir, "recordings")
	// 确保录制目录存在
	os.MkdirAll(recordingsDir, 0755)

	return &TTYDService{
		dataDir:        dataDir,
		processes:      make(map[string]*TTYDProcess),
		basePort:       7681,
		auditService:   auditService,
		sessionService: sessionService,
		recordingsDir:  recordingsDir,
	}
}

// StartTTYDSession 启动ttyd会话
func (ts *TTYDService) StartTTYDSession(server *models.Server, userID int, username string) (*TTYDProcess, error) {
	return ts.StartTTYDSessionWithAudit(server, userID, username, "", "")
}

// FindActiveSession 查找用户在指定服务器上的活跃会话
func (ts *TTYDService) FindActiveSession(userID, serverID int) (*TTYDProcess, bool) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	for _, process := range ts.processes {
		if process.UserID == userID && process.ServerID == serverID {
			return process, true
		}
	}
	return nil, false
}

// StartTTYDSessionWithAudit 启动ttyd会话并记录审计信息
func (ts *TTYDService) StartTTYDSessionWithAudit(server *models.Server, userID int, username, ipAddress, userAgent string) (*TTYDProcess, error) {
	// 首先检查是否有活跃会话可以复用
	if existingProcess, exists := ts.FindActiveSession(userID, server.ID); exists {
		log.Printf("复用现有ttyd会话: sessionID=%s, userID=%d, serverID=%d", existingProcess.SessionID, userID, server.ID)
		return existingProcess, nil
	}

	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// 生成会话ID
	sessionID := fmt.Sprintf("%s_%s_%d_%d", username, server.Name, userID, time.Now().Unix())

	// 查找可用端口
	port := ts.findAvailablePort()

	// 创建录制文件路径（暂时预留，后续实现应用层录制）
	timestamp := time.Now().Format("20060102_150405")
	recordingFileName := fmt.Sprintf("%s_%s_%s.cast", sessionID, timestamp, server.Name)
	recordingFilePath := filepath.Join(ts.recordingsDir, recordingFileName)

	// 组装 ttyd 启动参数（命令与参数分开传递给 ttyd）
	args := []string{
		"-p", fmt.Sprintf("%d", port),
		"-T", "xterm-256color",
		"-t", "enableZmodem=true",
		"-t", "enableTrzsz=true",
		"-W",
		"-b", "/proxy-terminal", // 设置基础路径以匹配代理
	}

	if server.AuthType == "password" {
		// 动态创建expect脚本文件
		expectScript, err := ts.createTempExpectScript(sessionID)
		if err != nil {
			return nil, fmt.Errorf("创建expect脚本失败: %v", err)
		}
		log.Printf("DEBUG: Expect script params: user=%s, pass=[REDACTED], host=%s, port=%d", server.Username, server.Host, server.Port)
		// 直接使用expect脚本，不依赖系统录制命令
		args = append(args,
			"expect",
			expectScript,
			server.Username,
			server.Password,
			server.Host,
			strconv.Itoa(server.Port),
		)
	} else if server.AuthType == "key" {
		// 创建临时密钥文件
		keyFile, err := ts.createTempKeyFile(server.PrivateKey, sessionID)
		if err != nil {
			return nil, fmt.Errorf("创建临时密钥文件失败: %v", err)
		}
		// 使用 ssh 作为命令，参数分开放置
		args = append(args,
			"ssh",
			"-i", keyFile,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			fmt.Sprintf("%s@%s", server.Username, server.Host),
			"-p", strconv.Itoa(server.Port),
		)
	} else {
		return nil, fmt.Errorf("不支持的认证类型: %s", server.AuthType)
	}

	// 创建上下文以便取消
	ctx, cancel := context.WithCancel(context.Background())

	// 启动真实的ttyd进程
	log.Printf("启动ttyd进程: port=%d, args=%v", port, args)
	cmd := exec.CommandContext(ctx, "ttyd", args...)
	log.Printf("启动ttyd进程时当前目录是： %v", cmd.Dir)

	// 设置环境变量
	env := os.Environ()

	cmd.Env = append(env,
		"TERM=xterm-256color",
		fmt.Sprintf("SESSION_ID=%s", sessionID),
	)

	// 设置进程组，便于管理子进程
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// 设置输出，便于调试
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		log.Printf("启动ttyd进程失败: %v, args=%v", err, args)
		return nil, fmt.Errorf("启动ttyd进程失败: %v", err)
	}

	// 等待ttyd端口可用
	if !ts.waitForPort(ctx, port, 5*time.Second) {
		cancel()
		log.Printf("等待ttyd端口 %d 超时", port)
		// 确保清理
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return nil, fmt.Errorf("等待ttyd端口 %d 超时", port)
	}

	log.Printf("ttyd进程启动成功: PID=%d, sessionID=%s", cmd.Process.Pid, sessionID)

	// 创建录制器
	recorder := NewSessionRecorder(sessionID, recordingFilePath, 80, 24)

	// 创建进程信息
	process := &TTYDProcess{
		SessionID:     sessionID,
		UserID:        userID,
		Username:      username,
		ServerID:      server.ID,
		ServerName:    server.Name, // 填充服务器名称
		Port:          port,
		Process:       cmd.Process,
		Cancel:        cancel,
		CreatedAt:     time.Now(),
		RecordingFile: recordingFilePath,
		Recorder:      recorder,
	}

	// 启动录制
	if err := recorder.Start(); err != nil {
		log.Printf("Failed to start recording: %v", err)
	}

	// 保存进程信息
	ts.processes[sessionID] = process

	// 启动监控协程
	go ts.monitorProcess(process, cmd)

	// 记录审计日志
	if ts.auditService != nil {
		go func() {
			if err := ts.auditService.LogTerminalStart(context.Background(), userID, server.ID, sessionID, ipAddress, userAgent); err != nil {
				log.Printf("Failed to log terminal start: %v", err)
			}
		}()
	}

	// 创建历史会话记录（同步执行，避免并发问题）
	if ts.sessionService != nil {
		// 稍后延迟创建，让主进程先完成启动
		time.Sleep(100 * time.Millisecond)
		if session, err := ts.sessionService.Create(userID, server.ID, ipAddress, recordingFileName); err != nil {
			log.Printf("Failed to create session record: %v", err)
		} else {
			// 保存数据库会话ID到进程信息中
			process.DBSessionID = session.ID
			log.Printf("Session record created: %s, recording file: %s", session.ID, recordingFileName)
		}
	}

	log.Printf("ttyd会话启动成功: sessionID=%s, port=%d", sessionID, port)
	return process, nil
}

// StopTTYDSession 停止ttyd会话
func (ts *TTYDService) StopTTYDSession(sessionID string) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	process, exists := ts.processes[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	// 取消上下文
	process.Cancel()

	// 杀死进程组
	if process.Process != nil {
		syscall.Kill(-process.Process.Pid, syscall.SIGTERM)

		// 等待进程结束，超时后强制杀死
		done := make(chan error, 1)
		go func() {
			_, err := process.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// 进程正常结束
		case <-time.After(5 * time.Second):
			// 超时，强制杀死
			syscall.Kill(-process.Process.Pid, syscall.SIGKILL)
		}
	}

	// 清理临时文件
	ts.cleanupTempFiles(sessionID)

	// 记录审计日志
	if ts.auditService != nil {
		go func() {
			if err := ts.auditService.LogTerminalEnd(context.Background(), sessionID, "manual_stop"); err != nil {
				log.Printf("Failed to log terminal end audit: %v", err)
			}
		}()
	}

	// 停止录制
	if process.Recorder != nil {
		if err := process.Recorder.Stop(); err != nil {
			log.Printf("Failed to stop recording: %v", err)
		}
	}

	// 更新数据库中的会话状态
	if ts.sessionService != nil && process.DBSessionID != "" {
		go func() {
			if err := ts.sessionService.Close(process.DBSessionID); err != nil {
				log.Printf("Failed to close session in database: %v", err)
			}
		}()
	}

	// 删除进程信息
	delete(ts.processes, sessionID)

	log.Printf("ttyd会话已停止: sessionID=%s", sessionID)
	return nil
}

// GetTTYDProcess 获取ttyd进程信息
func (ts *TTYDService) GetTTYDProcess(sessionID string) (*TTYDProcess, bool) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	process, exists := ts.processes[sessionID]
	return process, exists
}

// ListActiveSessions 列出活跃会话
func (ts *TTYDService) ListActiveSessions() []*TTYDProcess {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	sessions := make([]*TTYDProcess, 0, len(ts.processes))
	for _, process := range ts.processes {
		sessions = append(sessions, process)
	}
	return sessions
}

// CleanupExpiredSessions 清理过期会话
func (ts *TTYDService) CleanupExpiredSessions(maxAge time.Duration) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	now := time.Now()
	for sessionID, process := range ts.processes {
		if now.Sub(process.CreatedAt) > maxAge {
			log.Printf("清理过期会话: %s", sessionID)
			process.Cancel()
			if process.Process != nil {
				syscall.Kill(-process.Process.Pid, syscall.SIGTERM)
			}
			ts.cleanupTempFiles(sessionID)
			delete(ts.processes, sessionID)
		}
	}
}

// findAvailablePort 查找可用端口
func (ts *TTYDService) findAvailablePort() int {
	port := ts.basePort
	for {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			// 端口被占用，继续尝试下一个
			port++
			continue
		}
		// 端口可用，关闭监听器并返回端口
		listener.Close()
		ts.basePort = port + 1 // 更新基础端口，以便下次从新位置开始查找
		return port
	}
}

// createTempKeyFile 创建临时密钥文件
func (ts *TTYDService) createTempKeyFile(privateKey string, sessionID string) (string, error) {
	tempDir := filepath.Join(ts.dataDir, "temp_keys")
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return "", err
	}

	keyFile := filepath.Join(tempDir, fmt.Sprintf("key_%s", sessionID))
	if err := os.WriteFile(keyFile, []byte(privateKey), 0600); err != nil {
		return "", err
	}

	return keyFile, nil
}

// createTempExpectScript 创建临时expect脚本文件
func (ts *TTYDService) createTempExpectScript(sessionID string) (string, error) {
	// 确保临时目录存在
	tempDir := filepath.Join(ts.dataDir, "temp_keys")
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 创建临时expect脚本文件
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("ssh_connect_%s.exp", sessionID))
	if err := os.WriteFile(scriptFile, []byte(sshConnectExpScript), 0755); err != nil {
		return "", fmt.Errorf("写入expect脚本文件失败: %v", err)
	}

	log.Printf("创建临时expect脚本: %s", scriptFile)
	abs, err := filepath.Abs(scriptFile)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// cleanupTempFiles 清理临时文件
func (ts *TTYDService) cleanupTempFiles(sessionID string) {
	tempDir := filepath.Join(ts.dataDir, "temp_keys")

	// 清理密钥文件
	keyFile := filepath.Join(tempDir, fmt.Sprintf("key_%s", sessionID))
	os.Remove(keyFile)

	// 清理expect脚本文件
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("ssh_connect_%s.exp", sessionID))
	os.Remove(scriptFile)

	log.Printf("清理临时文件: %s", sessionID)
}

// monitorProcess 监控进程状态
func (ts *TTYDService) monitorProcess(process *TTYDProcess, cmd *exec.Cmd) {

	// 等待进程结束
	err := cmd.Wait()

	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// 清理进程信息
	if _, exists := ts.processes[process.SessionID]; exists {
		log.Printf("ttyd进程结束: sessionID=%s, error=%v", process.SessionID, err)

		// 记录终端结束
		reason := "ended"
		if err != nil {
			reason = "error"
		}
		if ts.auditService != nil {
			go func() {
				if err := ts.auditService.LogTerminalEnd(context.Background(), process.SessionID, reason); err != nil {
					log.Printf("Failed to log terminal end audit: %v", err)
				}
			}()
		}

		// 停止录制
		if process.Recorder != nil {
			if err := process.Recorder.Stop(); err != nil {
				log.Printf("Failed to stop recording: %v", err)
			}
		}

		// 更新数据库中的会话状态
		if ts.sessionService != nil && process.DBSessionID != "" {
			go func() {
				if err := ts.sessionService.Close(process.DBSessionID); err != nil {
					log.Printf("Failed to close session in database: %v", err)
				}
			}()
		}

		ts.cleanupTempFiles(process.SessionID)
		delete(ts.processes, process.SessionID)
	}

}

// waitForPort 等待端口可用
func (ts *TTYDService) waitForPort(ctx context.Context, port int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err == nil {
				conn.Close()
				return true
			}
		}
	}
}

// CleanupRecordings 清理录制文件
func (ts *TTYDService) CleanupRecordings(maxAge time.Duration) {
	recordingsDir := ts.recordingsDir

	// 遍历录制目录
	files, err := os.ReadDir(recordingsDir)
	if err != nil {
		log.Printf("Failed to read recordings directory: %v", err)
		return
	}

	now := time.Now()
	cleanedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 检查文件扩展名
		if !strings.HasSuffix(file.Name(), ".cast") {
			continue
		}

		filePath := filepath.Join(recordingsDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			log.Printf("Failed to get file info for %s: %v", filePath, err)
			continue
		}

		// 检查文件年龄
		if now.Sub(fileInfo.ModTime()) > maxAge {
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to remove old recording file %s: %v", filePath, err)
			} else {
				log.Printf("Removed old recording file: %s", filePath)
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		log.Printf("Cleaned up %d old recording files", cleanedCount)
	}
}

// GetRecordingsInfo 获取录制文件统计信息
func (ts *TTYDService) GetRecordingsInfo() (int, int64, error) {
	recordingsDir := ts.recordingsDir

	files, err := os.ReadDir(recordingsDir)
	if err != nil {
		return 0, 0, err
	}

	fileCount := 0
	var totalSize int64

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".cast") {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		fileCount++
		totalSize += fileInfo.Size()
	}

	return fileCount, totalSize, nil
}
