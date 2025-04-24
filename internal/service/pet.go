package service

import (
	"PetStore/internal/model"
	"PetStore/internal/repository"
	"context"
	"errors"
	"fmt"
)

type PetService interface {
	AddPet(ctx context.Context, pet *model.Pet) error
	UpdatePet(ctx context.Context, pet *model.Pet) error
	FindPetsByStatus(ctx context.Context, status []string) ([]*model.Pet, error)
	GetPetById(ctx context.Context, id int64) (*model.Pet, error)
	UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error
	DeletePet(ctx context.Context, id int64) error
	Upload(ctx context.Context, petID int64, imageURL string) error
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
	if pet.Status == "" {
		return errors.New("pet status is required")
	}
	_, err := s.repo.GetPetByID(ctx, pet.ID)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", pet.ID)
	}
	return s.repo.UpdatePet(ctx, pet)
}

func (s *petService) FindPetsByStatus(ctx context.Context, status []string) ([]*model.Pet, error) {
	if len(status) == 0 {
		return nil, errors.New("at least one status is required")
	}

	validStatus := map[string]bool{
		"available": true,
		"pending":   true,
		"sold":      true,
	}

	for _, s := range status {
		if !validStatus[s] {
			return nil, errors.New("invalid status value")
		}
	}

	return s.repo.FindPetsByStatus(ctx, status)
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

func (s *petService) Upload(ctx context.Context, petID int64, imageURL string) error {
	_, err := s.repo.GetPetByID(ctx, petID)
	if err != nil {
		return fmt.Errorf("pet with id %d not found", petID)
	}
	if petID <= 0 {
		return errors.New("invalid pet ID")
	}
	if imageURL == "" {
		return errors.New("URL image is required")
	}
	err = s.repo.UploadImage(ctx, petID, imageURL)
	if err != nil {
		return err
	}
	return nil
}
