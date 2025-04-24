package repository

import (
	"PetStore/internal/model"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pets interface {
	CreatePet(ctx context.Context, pet *model.Pet) error
	UpdatePet(ctx context.Context, pet *model.Pet) error
	DeletePet(ctx context.Context, id int64) error
	GetPetByID(ctx context.Context, id int64) (*model.Pet, error)
	FindPetsByStatus(ctx context.Context, status []string) ([]*model.Pet, error)
	UploadImage(ctx context.Context, petID int64, imageURL string) error
	UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Pets {
	return &Store{
		db: db,
	}
}

func (s *Store) CreatePet(ctx context.Context, pet *model.Pet) error {
	var categoryID int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO categories (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
		pet.Category.Name,
	).Scan(&categoryID)
	if err != nil {
		return err
	}

	err = s.db.QueryRow(
		ctx,
		`INSERT INTO pets (name, category_id, status, photo_urls)
         VALUES ($1, $2, $3, $4)
         RETURNING id`,
		pet.Name,
		pet.Category.ID,
		pet.Status,
		pet.ImageURL,
	).Scan(&pet.ID)
	if err != nil {
		return fmt.Errorf("save pet: %w", err)
	}

	for _, tag := range pet.Tags {
		err := s.db.QueryRow(
			ctx,
			`INSERT INTO tags (name) VALUES ($1) RETURNING id`,
			tag.Name,
		).Scan(&tag.ID)
		if err != nil {
			return fmt.Errorf("save tag: %w", err)
		}

		_, err = s.db.Exec(
			ctx,
			`INSERT INTO pet_to_tags (pet_id, tag_id) VALUES ($1, $2)`,
			pet.ID,
			tag.ID,
		)
		if err != nil {
			return fmt.Errorf("link tag: %w", err)
		}
	}

	return nil
}
func (s *Store) UpdatePet(ctx context.Context, pet *model.Pet) error {
	_, err := s.db.Exec(
		ctx,
		`INSERT INTO categories (id, name) 
             VALUES ($1, $2)
             ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
		pet.Category.ID,
		pet.Category.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	_, err = s.db.Exec(
		ctx,
		`UPDATE pets 
         SET name = $1, category_id = $2, status = $3, photo_urls = $4
         WHERE id = $5`,
		pet.Name,
		pet.Category.ID,
		pet.Status,
		pet.ImageURL,
		pet.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update pet: %w", err)
	}

	//Удаляем старые теги и добавляем новые
	_, err = s.db.Exec(ctx, `DELETE FROM pet_to_tags WHERE pet_id = $1`, pet.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old tags: %w", err)
	}

	for _, tag := range pet.Tags {
		err := s.db.QueryRow(
			ctx,
			`INSERT INTO tags (name) VALUES ($1) RETURNING id`,
			tag.Name,
		).Scan(&tag.ID)
		if err != nil {
			return fmt.Errorf("failed to save tag: %w", err)
		}

		_, err = s.db.Exec(
			ctx,
			`INSERT INTO pet_to_tags (pet_id, tag_id) VALUES ($1, $2)`,
			pet.ID,
			tag.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag: %w", err)
		}
	}

	return nil
}

func (s *Store) GetPetByID(ctx context.Context, id int64) (*model.Pet, error) {
	var pet model.Pet
	var categoryID *int64
	var categoryName *string

	err := s.db.QueryRow(
		ctx,
		`SELECT p.id, p.name, p.status, p.photo_urls, p.category_id, c.name
         FROM pets p
         LEFT JOIN categories c ON p.category_id = c.id
         WHERE p.id = $1`,
		id,
	).Scan(
		&pet.ID,
		&pet.Name,
		&pet.Status,
		&pet.ImageURL,
		&categoryID,
		&categoryName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pet: %w", err)
	}

	rows, err := s.db.Query(
		ctx,
		`SELECT t.id, t.name
         FROM tags t
         JOIN pet_to_tags pt ON t.id = pt.tag_id
         WHERE pt.pet_id = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		pet.Tags = append(pet.Tags, &tag)
	}

	return &pet, nil
}

func (s *Store) DeletePet(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM pets WHERE id = $1`, id)
	return err
}

func (s *Store) FindPetsByStatus(ctx context.Context, status []string) ([]*model.Pet, error) {
	var pets []*model.Pet

	rows, err := s.db.Query(
		ctx,
		`SELECT p.id, p.name, p.status, p.photo_urls, p.category_id, c.name
         FROM pets p
         LEFT JOIN categories c ON p.category_id = c.id
         WHERE p.status = $1`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find pets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pet model.Pet

		if err := rows.Scan(
			&pet.ID,
			&pet.Name,
			&pet.Status,
			&pet.ImageURL,
			&pet.Category.ID,
			&pet.Category.Name,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pet: %w", err)
		}
		tagrows, err := s.db.Query(
			ctx,
			`SELECT t.id, t.name
         FROM tags t
         JOIN pet_to_tags pt ON t.id = pt.tag_id
         WHERE pt.pet_id = $1`,
			pet.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}
		for tagrows.Next() {
			var tag model.Tag
			if err := tagrows.Scan(&tag.ID, &tag.Name); err != nil {
				tagrows.Close()
				return nil, fmt.Errorf("failed to scan tag: %w", err)
			}
			pet.Tags = append(pet.Tags, &tag)
		}
		tagrows.Close()
	}

	return pets, nil
}

func (s *Store) UploadImage(ctx context.Context, petID int64, imageURL string) error {
	_, err := s.db.Exec(
		ctx,
		`UPDATE pets 
         SET photo_urls = array_append(photo_urls, $1)
         WHERE id = $2`,
		imageURL,
		petID,
	)
	return err
}
func (s *Store) UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error {
	query := `UPDATE pets SET name = COALESCE(NULLIF($1, ''), name), status = COALESCE(NULLIF($2, ''), status) WHERE id = $3`
	_, err := s.db.Exec(ctx, query, name, status, id)
	return err
}
