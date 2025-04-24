package repository

import (
	"PetStore/internal/model"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserStore struct {
	db *pgxpool.Pool
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, username string) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUsersBatch(ctx context.Context, users []*model.User) error
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
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password failed: %w", err)
	}
	user.Password = string(hash)
	query := ` INSERT INTO users 
     (username, firstname, lastname, email, password_hash, phone, user_status)
     VALUES ($1, $2, $3, $4, $5, $6, $7); 
	RETURNING id;`
	err = s.db.QueryRow(ctx, query, user.UserName, user.FirstName, user.LastName, user.Email, user.Password, user.Phone, 0).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (s *UserStore) UpdateUser(ctx context.Context, user *model.User) error {
	query := ` UPDATE users SET
                  fist_name = $1,
                  last_name = $2,
                  email = $3,
                  password_hash = $4,
                  phone = $5,
 			  WHERE username = $6;`

	_, err := s.db.Exec(ctx, query, user.FirstName, user.LastName, user.Email, user.Password, user.Phone, user.UserName)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
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
	query := ` SELECT * FROM users WHERE username = $1;`
	err := s.db.QueryRow(ctx, query, username).Scan(user.ID, user.UserName, user.FirstName, user.LastName, user.Email, user.Password, user.Phone, user.UserStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (s *UserStore) CreateUsersBatch(ctx context.Context, users []*model.User) error {
	batch := &pgx.Batch{}

	for _, user := range users {
		hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		batch.Queue(
			`INSERT INTO users 
			 (username, first_name, last_name, email, password_hash, phone, user_status)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			user.UserName,
			user.FirstName,
			user.LastName,
			user.Email,
			string(hashedPwd),
			user.Phone,
			0,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()

	_, err := results.Exec()
	return err
}

func (s *UserStore) SetUserStatus(ctx context.Context, username string, status int32) error {
	query := ` UPDATE users SET status = $1 WHERE username = $2;`
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
	var token string
	err := s.db.QueryRow(ctx, "SELECT token FROM users WHERE username = $1;", username).Scan(&token)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token, nil
}
