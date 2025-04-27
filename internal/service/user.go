package service

import (
	"PetStore/internal/model"
	"PetStore/internal/repository"
	"context"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"
	"time"
)

type UserService interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, username string, updateData map[string]interface{}) error
	DeleteUser(ctx context.Context, username string) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUsersBatch(ctx context.Context, users []*model.User) ([]int64, error)
	LoginUser(ctx context.Context, username, password string) (string, error)
	LogoutUser() error
}
type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) CreateUser(ctx context.Context, user *model.User) error {
	existingUser, err := s.userRepo.GetUserByUsername(ctx, user.UserName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("user with username %s already exists", user.UserName)
	}

	return s.userRepo.CreateUser(ctx, user)
}

func (s *userService) UpdateUser(ctx context.Context, username string, updateData map[string]interface{}) error {
	_, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	return s.userRepo.UpdateUser(ctx, username, updateData)
}

func (s *userService) DeleteUser(ctx context.Context, username string) error {
	existingUser, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return fmt.Errorf("user with username %s not found", username)
	}
	return s.userRepo.DeleteUser(ctx, username)
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(ctx, username)
}

func (s *userService) CreateUsersBatch(ctx context.Context, users []*model.User) ([]int64, error) {
	for _, user := range users {
		existingUser, err := s.userRepo.GetUserByUsername(ctx, user.UserName)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check username failed: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("user with username %s already exists", user.UserName)
		}
	}
	return s.userRepo.CreateUsersBatch(ctx, users)
}
func (s *userService) LoginUser(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("User not found: %s", username)
			return "", fmt.Errorf("user not found")
		}
		log.Printf("Database error: %v", err)
		return "", fmt.Errorf("authentication failed")
	}

	log.Printf("Auth attempt. User: %s", username)
	log.Printf("DB Hash (raw): %x", []byte(user.Password))
	log.Printf("Input Password (raw): %x", []byte(password))

	if !strings.HasPrefix(user.Password, "$2a$") {
		log.Printf("Invalid hash format for user %s: %s", username, user.Password)
		return "", fmt.Errorf("authentication system error")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Password mismatch for user %s: %v", username, err)

		// Дополнительная проверка для отладки
		if user.Password == password {
			log.Printf("SECURITY ALERT: Password stored in plain text!")
			return "", fmt.Errorf("authentication failed - password not hashed")
		}

		return "", fmt.Errorf("authentication failed")
	}

	log.Printf("Password verified for user %s", username)

	existingToken, err := s.userRepo.GetToken(ctx, username)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Token check error: %v", err)
		return "", fmt.Errorf("authentication service unavailable")
	}

	if existingToken != "" {
		log.Printf("Returning existing token for user %s", username)
		return existingToken, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("JWT_SECRET is not configured")
		return "", fmt.Errorf("authentication service unavailable")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Token generation failed: %v", err)
		return "", fmt.Errorf("authentication service unavailable")
	}

	if err := s.userRepo.SaveToken(ctx, tokenString, username); err != nil {
		log.Printf("Token save failed: %v", err)
		return "", fmt.Errorf("authentication service unavailable")
	}

	if err := s.userRepo.SetUserStatus(ctx, username, 1); err != nil {
		log.Printf("Status update failed: %v", err)
	}

	log.Printf("New token generated for user %s", username)
	return tokenString, nil
}

func (s *userService) LogoutUser() error {
	return nil
}
