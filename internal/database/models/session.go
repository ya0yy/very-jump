package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Session 会话模型
type Session struct {
	ID            string     `json:"id" db:"id"`
	UserID        int        `json:"user_id" db:"user_id"`
	ServerID      int        `json:"server_id" db:"server_id"`
	StartTime     time.Time  `json:"start_time" db:"start_time"`
	EndTime       *time.Time `json:"end_time" db:"end_time"`
	Status        string     `json:"status" db:"status"`
	ClientIP      string     `json:"client_ip" db:"client_ip"`
	RecordingFile string     `json:"recording_file" db:"recording_file"`
	Username      string     `json:"username,omitempty"`    // 关联查询时使用
	ServerName    string     `json:"server_name,omitempty"` // 关联查询时使用
}

// SessionService 会话服务
type SessionService struct {
	db *sql.DB
}

// NewSessionService 创建会话服务
func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{db: db}
}

// Create 创建会话
func (s *SessionService) Create(userID, serverID int, clientIP, recordingFile string) (*Session, error) {
	sessionID := uuid.New().String()

	query := `
		INSERT INTO sessions (id, user_id, server_id, client_ip, recording_file) 
		VALUES (?, ?, ?, ?, ?) 
		RETURNING id, user_id, server_id, start_time, status, client_ip, recording_file
	`

	var session Session
	err := s.db.QueryRow(query, sessionID, userID, serverID, clientIP, recordingFile).Scan(
		&session.ID, &session.UserID, &session.ServerID, &session.StartTime,
		&session.Status, &session.ClientIP, &session.RecordingFile,
	)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// GetByID 根据ID获取会话
func (s *SessionService) GetByID(id string) (*Session, error) {
	query := `
		SELECT s.id, s.user_id, s.server_id, s.start_time, s.end_time, s.status, 
		       s.client_ip, s.recording_file, u.username, srv.name as server_name
		FROM sessions s
		LEFT JOIN users u ON s.user_id = u.id
		LEFT JOIN servers srv ON s.server_id = srv.id
		WHERE s.id = ?
	`

	var session Session
	err := s.db.QueryRow(query, id).Scan(
		&session.ID, &session.UserID, &session.ServerID, &session.StartTime,
		&session.EndTime, &session.Status, &session.ClientIP, &session.RecordingFile,
		&session.Username, &session.ServerName,
	)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// List 获取会话列表
func (s *SessionService) List(limit, offset int) ([]*Session, error) {
	query := `
		SELECT s.id, s.user_id, s.server_id, s.start_time, s.end_time, s.status, 
		       s.client_ip, s.recording_file, u.username, srv.name as server_name
		FROM sessions s
		LEFT JOIN users u ON s.user_id = u.id
		LEFT JOIN servers srv ON s.server_id = srv.id
		ORDER BY s.start_time DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		err := rows.Scan(&session.ID, &session.UserID, &session.ServerID,
			&session.StartTime, &session.EndTime, &session.Status,
			&session.ClientIP, &session.RecordingFile, &session.Username, &session.ServerName)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetByUserID 获取用户的会话列表
func (s *SessionService) GetByUserID(userID int, limit, offset int) ([]*Session, error) {
	query := `
		SELECT s.id, s.user_id, s.server_id, s.start_time, s.end_time, s.status, 
		       s.client_ip, s.recording_file, u.username, srv.name as server_name
		FROM sessions s
		LEFT JOIN users u ON s.user_id = u.id
		LEFT JOIN servers srv ON s.server_id = srv.id
		WHERE s.user_id = ?
		ORDER BY s.start_time DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		err := rows.Scan(&session.ID, &session.UserID, &session.ServerID,
			&session.StartTime, &session.EndTime, &session.Status,
			&session.ClientIP, &session.RecordingFile, &session.Username, &session.ServerName)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// UpdateStatus 更新会话状态
func (s *SessionService) UpdateStatus(id, status string) error {
	query := `UPDATE sessions SET status = ?, end_time = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.Exec(query, status, id)
	return err
}

// Close 关闭会话
func (s *SessionService) Close(id string) error {
	return s.UpdateStatus(id, "closed")
}

// GetActiveSessions 获取活跃会话数量
func (s *SessionService) GetActiveSessions() (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE status = 'active'`
	var count int
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}
