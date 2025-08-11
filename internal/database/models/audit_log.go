package models

import (
	"time"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	ServerID  int       `json:"server_id" db:"server_id"`
	Action    string    `json:"action" db:"action"`         // 操作类型：terminal_start, terminal_stop, command_execute 等
	Details   string    `json:"details" db:"details"`       // 操作详情 JSON
	SessionID string    `json:"session_id" db:"session_id"` // 终端会话ID
	IPAddress string    `json:"ip_address" db:"ip_address"` // 客户端IP
	UserAgent string    `json:"user_agent" db:"user_agent"` // 客户端信息
	Success   bool      `json:"success" db:"success"`       // 操作是否成功
	ErrorMsg  string    `json:"error_msg" db:"error_msg"`   // 错误信息
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TerminalSession 终端会话统计模型
type TerminalSession struct {
	ID           int        `json:"id" db:"id"`
	SessionID    string     `json:"session_id" db:"session_id"`
	UserID       int        `json:"user_id" db:"user_id"`
	ServerID     int        `json:"server_id" db:"server_id"`
	StartTime    time.Time  `json:"start_time" db:"start_time"`
	EndTime      *time.Time `json:"end_time" db:"end_time"`
	Duration     int        `json:"duration" db:"duration"`           // 持续时间（秒）
	CommandCount int        `json:"command_count" db:"command_count"` // 执行命令数量
	IPAddress    string     `json:"ip_address" db:"ip_address"`
	Status       string     `json:"status" db:"status"` // active, ended, error
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// SecurityAlert 安全告警模型
type SecurityAlert struct {
	ID          int        `json:"id" db:"id"`
	UserID      int        `json:"user_id" db:"user_id"`
	ServerID    int        `json:"server_id" db:"server_id"`
	AlertType   string     `json:"alert_type" db:"alert_type"`   // suspicious_command, multiple_login, unusual_time 等
	Severity    string     `json:"severity" db:"severity"`       // low, medium, high, critical
	Description string     `json:"description" db:"description"` // 告警描述
	Details     string     `json:"details" db:"details"`         // 详细信息 JSON
	IPAddress   string     `json:"ip_address" db:"ip_address"`
	SessionID   string     `json:"session_id" db:"session_id"`
	Resolved    bool       `json:"resolved" db:"resolved"`       // 是否已解决
	ResolvedBy  *int       `json:"resolved_by" db:"resolved_by"` // 解决人ID
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"` // 解决时间
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// AuditStatistics 审计统计信息
type AuditStatistics struct {
	TotalSessions    int `json:"total_sessions"`
	ActiveSessions   int `json:"active_sessions"`
	TotalCommands    int `json:"total_commands"`
	FailedLogins     int `json:"failed_logins"`
	SecurityAlerts   int `json:"security_alerts"`
	UnresolvedAlerts int `json:"unresolved_alerts"`
}
