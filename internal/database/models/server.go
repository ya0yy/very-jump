package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Server 服务器模型
type Server struct {
	ID            int        `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Host          string     `json:"host" db:"host"`
	Port          int        `json:"port" db:"port"`
	Username      string     `json:"username" db:"username"`
	AuthType      string     `json:"auth_type" db:"auth_type"`
	Password      string     `json:"-" db:"password"`
	PrivateKey    string     `json:"-" db:"private_key"`
	Description   string     `json:"description" db:"description"`
	TagsRaw       *string    `json:"-" db:"tags"` // 数据库存储的原始tags字符串
	Tags          []string   `json:"tags" db:"-"` // JSON返回的tags数组
	LastLoginTime *time.Time `json:"last_login_time" db:"last_login_time"`
	Status        string     `json:"status" db:"-"` // 不入库，实时检测
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ServerCreate 创建服务器请求
type ServerCreate struct {
	Name        string   `json:"name" binding:"required,min=1,max=100"`
	Host        string   `json:"host" binding:"required"`
	Port        int      `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    string   `json:"username" binding:"required"`
	AuthType    string   `json:"auth_type" binding:"required,oneof=password key"`
	Password    string   `json:"password" binding:"required_if=AuthType password"`
	PrivateKey  string   `json:"private_key" binding:"required_if=AuthType key"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// ServerUpdate 更新服务器请求
type ServerUpdate struct {
	Name        string   `json:"name" binding:"omitempty,min=1,max=100"`
	Host        string   `json:"host" binding:"omitempty"`
	Port        int      `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    string   `json:"username" binding:"omitempty"`
	AuthType    string   `json:"auth_type" binding:"omitempty,oneof=password key"`
	Password    string   `json:"password"`
	PrivateKey  string   `json:"private_key"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// ServerService 服务器服务
type ServerService struct {
	db *sql.DB
}

// NewServerService 创建服务器服务
func NewServerService(db *sql.DB) *ServerService {
	return &ServerService{db: db}
}

// tagsToString 将tags数组转换为字符串存储
func tagsToString(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	// 使用JSON格式存储，便于解析
	data, _ := json.Marshal(tags)
	return string(data)
}

// stringToTags 将字符串转换为tags数组
func stringToTags(tagsStr *string) []string {
	if tagsStr == nil || *tagsStr == "" {
		return []string{}
	}
	var tags []string
	json.Unmarshal([]byte(*tagsStr), &tags)
	return tags
}

// processTags 处理服务器的tags字段
func (s *Server) processTags() {
	s.Tags = stringToTags(s.TagsRaw)
}

// Create 创建服务器
func (s *ServerService) Create(req *ServerCreate) (*Server, error) {
	if req.Port == 0 {
		req.Port = 22
	}

	tagsStr := tagsToString(req.Tags)

	query := `
		INSERT INTO servers (name, host, port, username, auth_type, password, private_key, description, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, name, host, port, username, auth_type, description, tags, created_at, updated_at
	`

	var server Server
	err := s.db.QueryRow(query, req.Name, req.Host, req.Port, req.Username,
		req.AuthType, req.Password, req.PrivateKey, req.Description, tagsStr).Scan(
		&server.ID, &server.Name, &server.Host, &server.Port, &server.Username,
		&server.AuthType, &server.Description, &server.TagsRaw, &server.CreatedAt, &server.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	server.processTags()
	return &server, nil
}

// GetByID 根据ID获取服务器
func (s *ServerService) GetByID(id int) (*Server, error) {
	query := `SELECT id, name, host, port, username, auth_type, password, private_key, description, tags, last_login_time, created_at, updated_at FROM servers WHERE id = ?`

	var server Server
	err := s.db.QueryRow(query, id).Scan(
		&server.ID, &server.Name, &server.Host, &server.Port, &server.Username,
		&server.AuthType, &server.Password, &server.PrivateKey, &server.Description,
		&server.TagsRaw, &server.LastLoginTime, &server.CreatedAt, &server.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	server.processTags()
	return &server, nil
}

// List 获取服务器列表
func (s *ServerService) List(limit, offset int) ([]*Server, error) {
	query := `SELECT id, name, host, port, username, auth_type, description, tags, last_login_time, created_at, updated_at FROM servers ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*Server
	for rows.Next() {
		var server Server
		err := rows.Scan(&server.ID, &server.Name, &server.Host, &server.Port,
			&server.Username, &server.AuthType, &server.Description,
			&server.TagsRaw, &server.LastLoginTime, &server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
		server.processTags()
		servers = append(servers, &server)
	}

	return servers, nil
}

// GetByUserID 获取用户有权限的服务器列表
func (s *ServerService) GetByUserID(userID int, limit, offset int) ([]*Server, error) {
	query := `
		SELECT s.id, s.name, s.host, s.port, s.username, s.auth_type, s.description, s.tags, s.last_login_time, s.created_at, s.updated_at
		FROM servers s
		INNER JOIN user_server_permissions p ON s.id = p.server_id
		WHERE p.user_id = ?
		ORDER BY s.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*Server
	for rows.Next() {
		var server Server
		err := rows.Scan(&server.ID, &server.Name, &server.Host, &server.Port,
			&server.Username, &server.AuthType, &server.Description,
			&server.TagsRaw, &server.LastLoginTime, &server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
		server.processTags()
		servers = append(servers, &server)
	}

	return servers, nil
}

// Update 更新服务器
func (s *ServerService) Update(id int, req *ServerUpdate) (*Server, error) {
	server, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		server.Name = req.Name
	}
	if req.Host != "" {
		server.Host = req.Host
	}
	if req.Port != 0 {
		server.Port = req.Port
	}
	if req.Username != "" {
		server.Username = req.Username
	}
	if req.AuthType != "" {
		server.AuthType = req.AuthType
	}
	if req.Password != "" {
		server.Password = req.Password
	}
	if req.PrivateKey != "" {
		server.PrivateKey = req.PrivateKey
	}
	if req.Description != "" {
		server.Description = req.Description
	}
	if req.Tags != nil {
		server.Tags = req.Tags
		tagsStr := tagsToString(req.Tags)
		server.TagsRaw = &tagsStr
	}

	query := `
		UPDATE servers
		SET name = ?, host = ?, port = ?, username = ?, auth_type = ?, password = ?, private_key = ?, description = ?, tags = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	var tagsValue interface{}
	if server.TagsRaw != nil {
		tagsValue = *server.TagsRaw
	} else {
		tagsValue = nil
	}
	_, err = s.db.Exec(query, server.Name, server.Host, server.Port, server.Username,
		server.AuthType, server.Password, server.PrivateKey, server.Description, tagsValue, id)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// Delete 删除服务器
func (s *ServerService) Delete(id int) error {
	query := `DELETE FROM servers WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// GetServerCount 获取服务器总数
func (s *ServerService) GetServerCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM servers`
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateLastLoginTime 更新服务器上次登录时间
func (s *ServerService) UpdateLastLoginTime(id int) error {
	query := `UPDATE servers SET last_login_time = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// CheckServerStatus 检测服务器状态
func (s *Server) CheckServerStatus() string {
	timeout := 5 * time.Second
	address := fmt.Sprintf("%s:%d", s.Host, s.Port)

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return "unavailable"
	}
	defer conn.Close()

	return "available"
}

// CheckServerStatusByID 根据ID检测单个服务器状态
func (s *ServerService) CheckServerStatusByID(id int) (string, error) {
	server, err := s.GetByID(id)
	if err != nil {
		return "unavailable", err
	}

	status := server.CheckServerStatus()
	return status, nil
}

// CheckServersStatus 批量检测服务器状态
func (s *ServerService) CheckServersStatus(servers []*Server) {
	// 使用goroutine并发检测，提高效率
	done := make(chan bool, len(servers))

	for _, server := range servers {
		go func(srv *Server) {
			srv.Status = srv.CheckServerStatus()
			done <- true
		}(server)
	}

	// 等待所有检测完成
	for i := 0; i < len(servers); i++ {
		<-done
	}
}
