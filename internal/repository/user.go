package repository

import (
	"PetStore/internal/model"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserStore struct {
	db *pgxpool.Pool
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, username string, pos map[string]interface{}) error
	DeleteUser(ctx context.Context, username string) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUsersBatch(ctx context.Context, users []*model.User) ([]int64, error)
	SetUserStatus(ctx context.Context, username string, status int32) error
	SaveToken(ctx context.Context, token string, username string) error
	GetToken(ctx context.Context, username string) (string, error)
}

func NewUserStore(db *pgxpool.Pool) UserRepository {
	return &UserStore{
		db: db,
	}
}

func (s *UserStore) CreateUser(ctx context.Context, user *model.User) error {
	log.Printf("Creating user. Username: %s, Password: %s", user.UserName, user.Password)
	log.Printf("Original password: %s", user.Password) // Добавьте это
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing failed: %w", err)
	}
	log.Printf("Hashed password: %s", string(hashed)) // И это
	query := ` INSERT INTO users 
     (username, first_name, last_name, email, password_hash, phone, user_status)
     VALUES ($1, $2, $3, $4, $5, $6, $7) 
	RETURNING id`
	err = s.db.QueryRow(ctx, query, user.UserName, user.FirstName, user.LastName, user.Email, string(hashed), user.Phone, 0).Scan(&user.ID)
	if err != nil {
		log.Printf("DB error in CreateUser: %v", err) // Выведет настоящую ошибку
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *UserStore) UpdateUser(ctx context.Context, username string, updateData map[string]interface{}) error {
	if len(updateData) == 0 {
		return nil
	}

	var setClauses []string
	var args []interface{}
	argPos := 1

	// Поддерживаемые поля для обновления
	allowedFields := map[string]bool{
		"first_name":  true,
		"last_name":   true,
		"email":       true,
		"password":    true,
		"phone":       true,
		"user_status": true,
	}

	for field, value := range updateData {
		if !allowedFields[field] {
			continue
		}

		if field == "password" {
			hashed, err := bcrypt.GenerateFromPassword([]byte(value.(string)), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("password hashing failed: %w", err)
			}
			value = string(hashed)
			field = "password_hash" // Обновляем поле password_hash в БД
		}

		// Проверяем, что значение не nil
		if value == nil {
			continue
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argPos))
		args = append(args, value)
		argPos++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, username)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE username = $%d",
		strings.Join(setClauses, ", "),
		argPos,
	)

	result, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Проверяем, что была обновлена хотя бы одна запись
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (s *UserStore) DeleteUser(ctx context.Context, username string) error {
	query := ` DELETE FROM users WHERE username = $1;`
	_, err := s.db.Exec(ctx, query, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	query := `SELECT id, username, first_name, last_name, email, password_hash AS password, phone, user_status FROM users WHERE username = $1;`
	err := s.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.UserName,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.Phone,
		&user.UserStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (s *UserStore) CreateUsersBatch(ctx context.Context, users []*model.User) ([]int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	ids := make([]int64, len(users))

	for i, user := range users {
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO users 
             (username, first_name, last_name, email, password_hash, phone, user_status)
             VALUES ($1, $2, $3, $4, $5, $6, $7)
             RETURNING id`,
			user.UserName,
			user.FirstName,
			user.LastName,
			user.Email,
			string(hashedPwd),
			user.Phone,
			0,
		).Scan(&ids[i])

		if err != nil {
			return nil, fmt.Errorf("failed to insert user %s: %w", user.UserName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ids, nil
}

func (s *UserStore) SetUserStatus(ctx context.Context, username string, status int32) error {
	query := ` UPDATE users SET user_status = $1 WHERE username = $2;`
	_, err := s.db.Exec(ctx, query, status, username)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}
func (s *UserStore) SaveToken(ctx context.Context, token string, username string) error {
	query := ` UPDATE users SET token = $1 WHERE username = $2;`
	_, err := s.db.Exec(ctx, query, token, username)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}
func (s *UserStore) GetToken(ctx context.Context, username string) (string, error) {
	var token sql.NullString // Используем NullString вместо string
	err := s.db.QueryRow(ctx, "SELECT token FROM users WHERE username = $1", username).Scan(&token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	if !token.Valid {
		return "", nil // Возвращаем пустую строку для NULL значений
	}
	return token.String, nil
}
