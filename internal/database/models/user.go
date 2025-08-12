package models

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserCreate 创建用户请求
type UserCreate struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
}

// UserUpdate 更新用户请求
type UserUpdate struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=admin user"`
}

// UserService 用户服务
type UserService struct {
	db *sql.DB
}

// NewUserService 创建用户服务
func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// Create 创建用户
func (s *UserService) Create(req *UserCreate) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (username, password_hash, role) 
		VALUES (?, ?, ?) 
		RETURNING id, username, role, created_at, updated_at
	`

	var user User
	err = s.db.QueryRow(query, req.Username, string(hashedPassword), req.Role).Scan(
		&user.ID, &user.Username, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByID 根据ID获取用户
func (s *UserService) GetByID(id int) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE id = ?`

	var user User
	err := s.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (s *UserService) GetByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = ?`

	var user User
	err := s.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// List 获取用户列表
func (s *UserService) List(limit, offset int) ([]*User, error) {
	query := `SELECT id, username, role, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

// Update 更新用户
func (s *UserService) Update(id int, req *UserUpdate) (*User, error) {
	user, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hashedPassword)
	}

	query := `UPDATE users SET username = ?, password_hash = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = s.db.Exec(query, user.Username, user.PasswordHash, user.Role, id)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// Delete 删除用户
func (s *UserService) Delete(id int) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// ValidatePassword 验证密码
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// GetUserCount 获取用户总数
func (s *UserService) GetUserCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
