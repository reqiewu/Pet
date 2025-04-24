package service

import (
	"PetStore/internal/model"
	"PetStore/internal/repository"
	"context"
	"errors"
)

type StoreService interface {
	GetInventory(ctx context.Context) (map[string]int64, error)
	PlaceOrder(ctx context.Context, order *model.Order) error
	GetOrderById(ctx context.Context, id int64) (*model.Order, error)
	DeleteOrder(ctx context.Context, id int64) error
}

type storeService struct {
	repo repository.StoreRepository
}

func NewStoreService(repo repository.StoreRepository) StoreService {
	return &storeService{repo: repo}
}

func (s *storeService) GetInventory(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetInventory(ctx)
}

func (s *storeService) PlaceOrder(ctx context.Context, order *model.Order) error {
	if order.UserID <= 0 {
		return errors.New("неверный ID пользователя")
		if order.PetID <= 0 {
			return errors.New("invalid pet ID")
		}
		if order.Quantity <= 0 {
			return errors.New("quantity must be positive")
		}
		if order.Status == "" {
			return errors.New("status is required")
		}

		return s.repo.PlaceOrder(ctx, order)
	}
}

func (s *storeService) GetOrderById(ctx context.Context, id int64) (*model.Order, error) {
	if id <= 0 {
		return nil, errors.New("invalid order ID")
	}
	return s.repo.GetOrderById(ctx, id)
}

func (s *storeService) DeleteOrder(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid order ID")
	}
	return s.repo.DeleteOrder(ctx, id)
}
