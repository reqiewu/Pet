package service

import (
	"PetStore/internal/model"
	"PetStore/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
)

type PetService interface {
	AddPet(ctx context.Context, pet *model.Pet) error
	UpdatePet(ctx context.Context, pet *model.Pet) error
	FindPetsByStatus(ctx context.Context, status string) ([]*model.Pet, error)
	GetPetById(ctx context.Context, id int64) (*model.Pet, error)
	UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error
	DeletePet(ctx context.Context, id int64) error
	Upload(ctx context.Context, petID int64, imageURLs []string) error
	DebugPets(ctx context.Context) error
}

type petService struct {
	repo repository.Pets
}

func NewPetService(repo repository.Pets) PetService {
	return &petService{repo: repo}
}

func (s *petService) AddPet(ctx context.Context, pet *model.Pet) error {
	if pet.Name == "" {
		return errors.New("pet name is required")
	}
	if pet.Status == "" {
		return errors.New("pet status is required")
	}

	return s.repo.CreatePet(ctx, pet)
}

func (s *petService) UpdatePet(ctx context.Context, pet *model.Pet) error {
	if pet.ID == 0 {
		return errors.New("pet ID is required")
	}
	if pet.Name == "" {
		return errors.New("pet name is required")
	}
	_, err := s.repo.GetPetByID(ctx, pet.ID)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", pet.ID)
	}
	return s.repo.UpdatePet(ctx, pet)
}

func (s *petService) FindPetsByStatus(ctx context.Context, status string) ([]*model.Pet, error) {
	fmt.Printf("Service: received status: %s\n", status)

	// Валидация статуса
	statuses := strings.Split(status, ",")
	for _, s := range statuses {
		if s != "available" && s != "pending" && s != "sold" {
			fmt.Printf("Service: invalid status: %s\n", s)
			return nil, fmt.Errorf("invalid status: %s", s)
		}
	}

	fmt.Printf("Service: calling repository with status: %s\n", status)
	pets, err := s.repo.FindPetsByStatus(ctx, status)
	if err != nil {
		fmt.Printf("Service: repository error: %v\n", err)
		return nil, fmt.Errorf("failed to find pets: %w", err)
	}

	fmt.Printf("Service: found %d pets\n", len(pets))
	return pets, nil
}

func (s *petService) GetPetById(ctx context.Context, id int64) (*model.Pet, error) {
	if id <= 0 {
		return nil, errors.New("invalid pet ID")
	}

	return s.repo.GetPetByID(ctx, id)
}

func (s *petService) UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error {
	if id <= 0 {
		return errors.New("invalid pet ID")
	}
	if name == "" && status == "" {
		return errors.New("either name or status must be provided")
	}
	_, err := s.repo.GetPetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", id)
	}

	return s.repo.UpdatePetWithForm(ctx, id, name, status)
}

func (s *petService) DeletePet(ctx context.Context, id int64) error {
	_, err := s.repo.GetPetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", id)
	}
	if id <= 0 {
		return errors.New("invalid pet ID")
	}

	return s.repo.DeletePet(ctx, id)
}

func (s *petService) Upload(ctx context.Context, petID int64, imageURLs []string) error {
	if petID <= 0 {
		return errors.New("invalid pet ID")
	}
	if len(imageURLs) == 0 {
		return errors.New("at least one image URL is required")
	}

	// Проверяем существование питомца
	_, err := s.repo.GetPetByID(ctx, petID)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", petID)
	}

	return s.repo.UploadImage(ctx, petID, imageURLs)
}

func (s *petService) DebugPets(ctx context.Context) error {
	return s.repo.DebugPets(ctx)
}
