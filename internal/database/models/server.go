package models

import (
	"database/sql"
	"time"
)

// Server 服务器模型
type Server struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Host        string    `json:"host" db:"host"`
	Port        int       `json:"port" db:"port"`
	Username    string    `json:"username" db:"username"`
	AuthType    string    `json:"auth_type" db:"auth_type"`
	Password    string    `json:"-" db:"password"`
	PrivateKey  string    `json:"-" db:"private_key"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ServerCreate 创建服务器请求
type ServerCreate struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    string `json:"username" binding:"required"`
	AuthType    string `json:"auth_type" binding:"required,oneof=password key"`
	Password    string `json:"password" binding:"required_if=AuthType password"`
	PrivateKey  string `json:"private_key" binding:"required_if=AuthType key"`
	Description string `json:"description"`
}

// ServerUpdate 更新服务器请求
type ServerUpdate struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Host        string `json:"host" binding:"omitempty"`
	Port        int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username    string `json:"username" binding:"omitempty"`
	AuthType    string `json:"auth_type" binding:"omitempty,oneof=password key"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	Description string `json:"description"`
}

// ServerService 服务器服务
type ServerService struct {
	db *sql.DB
}

// NewServerService 创建服务器服务
func NewServerService(db *sql.DB) *ServerService {
	return &ServerService{db: db}
}

// Create 创建服务器
func (s *ServerService) Create(req *ServerCreate) (*Server, error) {
	if req.Port == 0 {
		req.Port = 22
	}

	query := `
		INSERT INTO servers (name, host, port, username, auth_type, password, private_key, description) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) 
		RETURNING id, name, host, port, username, auth_type, description, created_at, updated_at
	`

	var server Server
	err := s.db.QueryRow(query, req.Name, req.Host, req.Port, req.Username,
		req.AuthType, req.Password, req.PrivateKey, req.Description).Scan(
		&server.ID, &server.Name, &server.Host, &server.Port, &server.Username,
		&server.AuthType, &server.Description, &server.CreatedAt, &server.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &server, nil
}

// GetByID 根据ID获取服务器
func (s *ServerService) GetByID(id int) (*Server, error) {
	query := `SELECT id, name, host, port, username, auth_type, password, private_key, description, created_at, updated_at FROM servers WHERE id = ?`

	var server Server
	err := s.db.QueryRow(query, id).Scan(
		&server.ID, &server.Name, &server.Host, &server.Port, &server.Username,
		&server.AuthType, &server.Password, &server.PrivateKey, &server.Description,
		&server.CreatedAt, &server.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &server, nil
}

// List 获取服务器列表
func (s *ServerService) List(limit, offset int) ([]*Server, error) {
	query := `SELECT id, name, host, port, username, auth_type, description, created_at, updated_at FROM servers ORDER BY created_at DESC LIMIT ? OFFSET ?`

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
			&server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, &server)
	}

	return servers, nil
}

// GetByUserID 获取用户有权限的服务器列表
func (s *ServerService) GetByUserID(userID int, limit, offset int) ([]*Server, error) {
	query := `
		SELECT s.id, s.name, s.host, s.port, s.username, s.auth_type, s.description, s.created_at, s.updated_at 
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
			&server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
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

	query := `
		UPDATE servers 
		SET name = ?, host = ?, port = ?, username = ?, auth_type = ?, password = ?, private_key = ?, description = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`
	_, err = s.db.Exec(query, server.Name, server.Host, server.Port, server.Username,
		server.AuthType, server.Password, server.PrivateKey, server.Description, id)
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
