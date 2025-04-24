package service

import (
	"PetStore/internal/model"
	"PetStore/internal/repository"
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"os"
	"time"
)

type UserService interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, username string) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUsersBatch(ctx context.Context, users []*model.User) error
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
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("user with username %s already exists", user.UserName)
	}
	return s.userRepo.CreateUser(ctx, user)
}

func (s *userService) UpdateUser(ctx context.Context, user *model.User) error {
	existingUser, err := s.userRepo.GetUserByUsername(ctx, user.UserName)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return fmt.Errorf("user with username %s not found", user.UserName)
	}
	return s.userRepo.UpdateUser(ctx, user)
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

func (s *userService) CreateUsersBatch(ctx context.Context, users []*model.User) error {
	for _, user := range users {
		existingUser, err := s.userRepo.GetUserByUsername(ctx, user.UserName)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return fmt.Errorf("user with username %s already exists", user.UserName)
		}
	}
	return s.userRepo.CreateUsersBatch(ctx, users)
}

func (s *userService) LoginUser(ctx context.Context, username, password string) (string, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("user with username %s not found", username)
	}
	if user == nil {
		return "", fmt.Errorf("user with username %s not found", username)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid username or password")
	}
	existingToken, err := s.userRepo.GetToken(ctx, username)
	if err != nil {
		return "", fmt.Errorf("failed to check db on tokens")
	}
	if existingToken != "" {
		return existingToken, nil
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", fmt.Errorf("failed to load jwt token")
	}
	err = s.userRepo.SetUserStatus(ctx, username, 1)
	if err != nil {
		return "", fmt.Errorf("failed to set user status")
	}
	return tokenString, nil
}

func (s *userService) LogoutUser() error {
	return nil
}
