package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"very-jump/internal/database/models"
)

// AuditService 审计服务
type AuditService struct {
	db *sql.DB
}

// NewAuditService 创建审计服务实例
func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

// LogAction 记录操作审计日志
func (s *AuditService) LogAction(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, server_id, action, details, session_id, ip_address, user_agent, success, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		log.UserID, log.ServerID, log.Action, log.Details, log.SessionID,
		log.IPAddress, log.UserAgent, log.Success, log.ErrorMsg)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

// LogTerminalStart 记录终端启动
func (s *AuditService) LogTerminalStart(ctx context.Context, userID, serverID int, sessionID, ipAddress, userAgent string) error {
	details := map[string]interface{}{
		"session_id": sessionID,
		"timestamp":  time.Now().UTC(),
	}
	detailsJSON, _ := json.Marshal(details)

	auditLog := &models.AuditLog{
		UserID:    userID,
		ServerID:  serverID,
		Action:    "terminal_start",
		Details:   string(detailsJSON),
		SessionID: sessionID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	}

	// 记录审计日志
	if err := s.LogAction(ctx, auditLog); err != nil {
		log.Printf("Failed to log terminal start: %v", err)
	}

	// 创建会话记录
	return s.CreateTerminalSession(ctx, sessionID, userID, serverID, ipAddress)
}

// LogTerminalEnd 记录终端结束
func (s *AuditService) LogTerminalEnd(ctx context.Context, sessionID, reason string) error {
	// 更新会话状态
	if err := s.EndTerminalSession(ctx, sessionID, reason); err != nil {
		log.Printf("Failed to end terminal session: %v", err)
	}

	// 获取会话信息用于审计日志
	session, err := s.GetTerminalSession(ctx, sessionID)
	if err != nil {
		log.Printf("Failed to get terminal session for audit: %v", err)
		return nil // 不影响主流程
	}

	details := map[string]interface{}{
		"session_id": sessionID,
		"reason":     reason,
		"duration":   session.Duration,
		"commands":   session.CommandCount,
		"timestamp":  time.Now().UTC(),
	}
	detailsJSON, _ := json.Marshal(details)

	auditLog := &models.AuditLog{
		UserID:    session.UserID,
		ServerID:  session.ServerID,
		Action:    "terminal_end",
		Details:   string(detailsJSON),
		SessionID: sessionID,
		IPAddress: session.IPAddress,
		Success:   true,
	}

	return s.LogAction(ctx, auditLog)
}

// CreateTerminalSession 创建终端会话记录
func (s *AuditService) CreateTerminalSession(ctx context.Context, sessionID string, userID, serverID int, ipAddress string) error {
	query := `
		INSERT INTO terminal_sessions (session_id, user_id, server_id, start_time, ip_address, status)
		VALUES (?, ?, ?, ?, ?, 'active')
	`
	_, err := s.db.ExecContext(ctx, query, sessionID, userID, serverID, time.Now().UTC(), ipAddress)
	if err != nil {
		return fmt.Errorf("failed to create terminal session: %w", err)
	}
	return nil
}

// EndTerminalSession 结束终端会话
func (s *AuditService) EndTerminalSession(ctx context.Context, sessionID, reason string) error {
	// 计算持续时间
	query := `
		UPDATE terminal_sessions 
		SET end_time = ?, 
		    duration = CAST((julianday(?) - julianday(start_time)) * 86400 AS INTEGER),
		    status = ?,
		    updated_at = ?
		WHERE session_id = ? AND status = 'active'
	`
	now := time.Now().UTC()
	status := "ended"
	if reason == "error" {
		status = "error"
	}

	_, err := s.db.ExecContext(ctx, query, now, now, status, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end terminal session: %w", err)
	}
	return nil
}

// GetTerminalSession 获取终端会话信息
func (s *AuditService) GetTerminalSession(ctx context.Context, sessionID string) (*models.TerminalSession, error) {
	query := `
		SELECT id, session_id, user_id, server_id, start_time, end_time, duration, 
		       command_count, ip_address, status, created_at, updated_at
		FROM terminal_sessions 
		WHERE session_id = ?
	`
	row := s.db.QueryRowContext(ctx, query, sessionID)

	session := &models.TerminalSession{}
	err := row.Scan(
		&session.ID, &session.SessionID, &session.UserID, &session.ServerID,
		&session.StartTime, &session.EndTime, &session.Duration, &session.CommandCount,
		&session.IPAddress, &session.Status, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal session: %w", err)
	}
	return session, nil
}

// CreateSecurityAlert 创建安全告警
func (s *AuditService) CreateSecurityAlert(ctx context.Context, alert *models.SecurityAlert) error {
	query := `
		INSERT INTO security_alerts (user_id, server_id, alert_type, severity, description, details, ip_address, session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		alert.UserID, alert.ServerID, alert.AlertType, alert.Severity,
		alert.Description, alert.Details, alert.IPAddress, alert.SessionID)

	if err != nil {
		return fmt.Errorf("failed to create security alert: %w", err)
	}

	log.Printf("Security Alert [%s]: %s (User: %d, Server: %d)",
		alert.Severity, alert.Description, alert.UserID, alert.ServerID)
	return nil
}

// CheckSuspiciousCommand 检查可疑命令
func (s *AuditService) CheckSuspiciousCommand(ctx context.Context, userID, serverID int, sessionID, command, ipAddress string) {
	suspiciousPatterns := []string{
		"rm -rf", "dd if=", ":(){ :|:& };:", // 危险命令
		"wget", "curl", "nc ", "netcat", // 网络下载工具
		"sudo su", "su -", "sudo -i", // 权限提升
		"passwd", "useradd", "userdel", // 用户管理
		"iptables", "ufw", "firewall", // 防火墙操作
		"systemctl", "service", // 系统服务
		"crontab", "at ", "nohup", // 计划任务
	}

	command = strings.ToLower(strings.TrimSpace(command))
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(command, pattern) {
			details := map[string]interface{}{
				"command":    command,
				"pattern":    pattern,
				"timestamp":  time.Now().UTC(),
				"session_id": sessionID,
			}
			detailsJSON, _ := json.Marshal(details)

			alert := &models.SecurityAlert{
				UserID:      userID,
				ServerID:    serverID,
				AlertType:   "suspicious_command",
				Severity:    s.getSeverityForCommand(pattern),
				Description: fmt.Sprintf("Suspicious command detected: %s", command),
				Details:     string(detailsJSON),
				IPAddress:   ipAddress,
				SessionID:   sessionID,
			}

			if err := s.CreateSecurityAlert(ctx, alert); err != nil {
				log.Printf("Failed to create security alert: %v", err)
			}
			break
		}
	}
}

// getSeverityForCommand 根据命令获取严重级别
func (s *AuditService) getSeverityForCommand(pattern string) string {
	highRisk := []string{"rm -rf", "dd if=", ":(){ :|:& };:", "sudo su", "su -"}
	mediumRisk := []string{"wget", "curl", "passwd", "useradd", "iptables"}

	for _, cmd := range highRisk {
		if pattern == cmd {
			return "high"
		}
	}
	for _, cmd := range mediumRisk {
		if pattern == cmd {
			return "medium"
		}
	}
	return "low"
}

// GetAuditLogs 获取审计日志列表
func (s *AuditService) GetAuditLogs(ctx context.Context, userID *int, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, user_id, server_id, action, details, session_id, ip_address, 
		       user_agent, success, error_msg, created_at
		FROM audit_logs
	`
	args := []interface{}{}

	if userID != nil {
		query += " WHERE user_id = ?"
		args = append(args, *userID)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.ServerID, &log.Action, &log.Details,
			&log.SessionID, &log.IPAddress, &log.UserAgent, &log.Success,
			&log.ErrorMsg, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetSecurityAlerts 获取安全告警列表
func (s *AuditService) GetSecurityAlerts(ctx context.Context, resolved *bool, limit, offset int) ([]*models.SecurityAlert, error) {
	query := `
		SELECT id, user_id, server_id, alert_type, severity, description, details,
		       ip_address, session_id, resolved, resolved_by, resolved_at, created_at
		FROM security_alerts
	`
	args := []interface{}{}

	if resolved != nil {
		query += " WHERE resolved = ?"
		args = append(args, *resolved)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query security alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.SecurityAlert
	for rows.Next() {
		alert := &models.SecurityAlert{}
		err := rows.Scan(
			&alert.ID, &alert.UserID, &alert.ServerID, &alert.AlertType, &alert.Severity,
			&alert.Description, &alert.Details, &alert.IPAddress, &alert.SessionID,
			&alert.Resolved, &alert.ResolvedBy, &alert.ResolvedAt, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan security alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetAuditStatistics 获取审计统计信息
func (s *AuditService) GetAuditStatistics(ctx context.Context) (*models.AuditStatistics, error) {
	stats := &models.AuditStatistics{}

	// 总会话数
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM terminal_sessions").Scan(&stats.TotalSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions: %w", err)
	}

	// 活跃会话数
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM terminal_sessions WHERE status = 'active'").Scan(&stats.ActiveSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// 总命令数
	err = s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(command_count), 0) FROM terminal_sessions").Scan(&stats.TotalCommands)
	if err != nil {
		return nil, fmt.Errorf("failed to get total commands: %w", err)
	}

	// 失败登录数（过去24小时）
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM audit_logs WHERE action = 'login' AND success = FALSE AND created_at > datetime('now', '-24 hours')").
		Scan(&stats.FailedLogins)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed logins: %w", err)
	}

	// 安全告警数
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM security_alerts").Scan(&stats.SecurityAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to get security alerts: %w", err)
	}

	// 未解决告警数
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM security_alerts WHERE resolved = FALSE").Scan(&stats.UnresolvedAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved alerts: %w", err)
	}

	return stats, nil
}








