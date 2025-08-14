package models

import (
	"database/sql"
	"time"
)

// Credential 登录凭证模型
type Credential struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Type        string    `json:"type" db:"type"` // "password" 或 "key"
	Username    string    `json:"username" db:"username"`
	Password    string    `json:"-" db:"password"`     // 不在JSON中返回
	PrivateKey  string    `json:"-" db:"private_key"`  // 不在JSON中返回
	KeyPassword string    `json:"-" db:"key_password"` // 不在JSON中返回
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CredentialCreate 创建登录凭证请求
type CredentialCreate struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Type        string `json:"type" binding:"required,oneof=password key"`
	Username    string `json:"username" binding:"required,min=1,max=100"`
	Password    string `json:"password" binding:"required_if=Type password"`
	PrivateKey  string `json:"private_key" binding:"required_if=Type key"`
	KeyPassword string `json:"key_password"` // 可选
}

// CredentialUpdate 更新登录凭证请求
type CredentialUpdate struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Type        string `json:"type" binding:"omitempty,oneof=password key"`
	Username    string `json:"username" binding:"omitempty,min=1,max=100"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	KeyPassword string `json:"key_password"`
}

// CredentialService 登录凭证服务
type CredentialService struct {
	db *sql.DB
}

// NewCredentialService 创建登录凭证服务
func NewCredentialService(db *sql.DB) *CredentialService {
	return &CredentialService{db: db}
}

// Create 创建登录凭证
func (s *CredentialService) Create(req *CredentialCreate) (*Credential, error) {
	query := `
		INSERT INTO credentials (name, type, username, password, private_key, key_password) 
		VALUES (?, ?, ?, ?, ?, ?) 
		RETURNING id, name, type, username, created_at, updated_at
	`

	var credential Credential
	err := s.db.QueryRow(query, req.Name, req.Type, req.Username,
		req.Password, req.PrivateKey, req.KeyPassword).Scan(
		&credential.ID, &credential.Name, &credential.Type, &credential.Username,
		&credential.CreatedAt, &credential.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &credential, nil
}

// GetByID 根据ID获取登录凭证
func (s *CredentialService) GetByID(id int) (*Credential, error) {
	query := `SELECT id, name, type, username, password, private_key, key_password, created_at, updated_at FROM credentials WHERE id = ?`

	var credential Credential
	err := s.db.QueryRow(query, id).Scan(
		&credential.ID, &credential.Name, &credential.Type, &credential.Username,
		&credential.Password, &credential.PrivateKey, &credential.KeyPassword,
		&credential.CreatedAt, &credential.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &credential, nil
}

// List 获取登录凭证列表
func (s *CredentialService) List(limit, offset int) ([]*Credential, error) {
	query := `SELECT id, name, type, username, created_at, updated_at FROM credentials ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []*Credential
	for rows.Next() {
		var credential Credential
		err := rows.Scan(&credential.ID, &credential.Name, &credential.Type,
			&credential.Username, &credential.CreatedAt, &credential.UpdatedAt)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

// Update 更新登录凭证
func (s *CredentialService) Update(id int, req *CredentialUpdate) (*Credential, error) {
	credential, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		credential.Name = req.Name
	}
	if req.Type != "" {
		credential.Type = req.Type
	}
	if req.Username != "" {
		credential.Username = req.Username
	}
	if req.Password != "" {
		credential.Password = req.Password
	}
	if req.PrivateKey != "" {
		credential.PrivateKey = req.PrivateKey
	}
	if req.KeyPassword != "" {
		credential.KeyPassword = req.KeyPassword
	}

	query := `
		UPDATE credentials 
		SET name = ?, type = ?, username = ?, password = ?, private_key = ?, key_password = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?
	`
	_, err = s.db.Exec(query, credential.Name, credential.Type, credential.Username,
		credential.Password, credential.PrivateKey, credential.KeyPassword, id)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// Delete 删除登录凭证
func (s *CredentialService) Delete(id int) error {
	query := `DELETE FROM credentials WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// GetCredentialCount 获取登录凭证总数
func (s *CredentialService) GetCredentialCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM credentials`
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
