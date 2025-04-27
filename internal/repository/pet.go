package repository

import (
	"PetStore/internal/model"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type Pets interface {
	CreatePet(ctx context.Context, pet *model.Pet) error
	UpdatePet(ctx context.Context, pet *model.Pet) error
	DeletePet(ctx context.Context, id int64) error
	GetPetByID(ctx context.Context, id int64) (*model.Pet, error)
	FindPetsByStatus(ctx context.Context, status string) ([]*model.Pet, error)
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
	// Проверка обязательных полей
	if pet == nil {
		return fmt.Errorf("pet is nil")
	}
	if pet.Name == "" {
		return fmt.Errorf("pet name is required")
	}

	// Установка дефолтных значений
	if pet.Status != "available" && pet.Status != "pending" && pet.Status != "sold" {
		pet.Status = "available"
	}
	if pet.Category == nil {
		pet.Category = &model.Category{Name: "unknown"}
	}

	// Вставка или получение категории
	var categoryID int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO pet_categories (name) VALUES ($1)
         ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
         RETURNING id`,
		pet.Category.Name,
	).Scan(&categoryID)
	if err != nil {
		return fmt.Errorf("failed to save category: %w", err)
	}
	pet.Category.ID = categoryID

	// Вставка питомца
	var petID int64
	err = s.db.QueryRow(ctx,
		`INSERT INTO pets (name, category_id, status, photo_urls)
         VALUES ($1, $2, $3, $4)
         RETURNING id`,
		pet.Name,
		pet.Category.ID,
		pet.Status,
		pq.Array(pet.ImageURL), // Используем pq.Array для массива
	).Scan(&petID)
	if err != nil {
		return fmt.Errorf("failed to save pet: %w", err)
	}
	pet.ID = petID

	// Обработка тегов
	if len(pet.Tags) > 0 {
		for _, tag := range pet.Tags {
			if tag == nil {
				continue
			}

			// Вставка или получение тега
			var tagID int64
			err := s.db.QueryRow(ctx,
				`INSERT INTO pet_tags (name) VALUES ($1)
                 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                 RETURNING id`,
				tag.Name,
			).Scan(&tagID)
			if err != nil {
				return fmt.Errorf("failed to save tag: %w", err)
			}
			tag.ID = tagID

			// Связь питомца с тегом
			_, err = s.db.Exec(ctx,
				`INSERT INTO pet_to_tags (pet_id, tag_id) VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`,
				pet.ID,
				tag.ID,
			)
			if err != nil {
				return fmt.Errorf("failed to link tag: %w", err)
			}
		}
	}

	return nil
}
func (s *Store) UpdatePet(ctx context.Context, pet *model.Pet) error {
	// Проверка обязательных полей
	if pet == nil {
		return fmt.Errorf("pet cannot be nil")
	}
	if pet.ID == 0 {
		return fmt.Errorf("pet ID is required")
	}

	// 1. Обновляем категорию (если указана)
	if pet.Category != nil {
		if pet.Category.Name != "" {
			err := s.db.QueryRow(
				ctx,
				`INSERT INTO pet_categories (name) 
                 VALUES ($1)
                 ON CONFLICT (name) DO UPDATE 
                 SET name = EXCLUDED.name 
                 RETURNING id`,
				pet.Category.Name,
			).Scan(&pet.Category.ID)
			if err != nil {
				return fmt.Errorf("failed to update category: %w", err)
			}
		}
	}

	// 2. Валидация статуса
	if pet.Status != "available" && pet.Status != "pending" && pet.Status != "sold" {
		pet.Status = "available"
	}

	// 3. Обновляем основную информацию о питомце
	result, err := s.db.Exec(
		ctx,
		`UPDATE pets 
         SET name = COALESCE($1, name), 
             status = COALESCE($2, status),
             photo_urls = COALESCE($3, photo_urls),
             category_id = COALESCE($4, category_id)
         WHERE id = $5`,
		pet.Name,
		pet.Status,
		pq.Array(pet.ImageURL),
		pet.Category.ID, // Используем метод-геттер
		pet.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update pet: %w", err)
	}

	// Проверяем, что запрос действительно обновил запись
	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return fmt.Errorf("pet with ID %d not found", pet.ID)
	}

	// 4. Обновляем теги (если они указаны)
	if pet.Tags != nil {
		// Удаляем старые связи
		_, err = s.db.Exec(ctx, `DELETE FROM pet_to_tags WHERE pet_id = $1`, pet.ID)
		if err != nil {
			return fmt.Errorf("failed to delete old tags: %w", err)
		}

		// Добавляем новые теги
		for _, tag := range pet.Tags {
			if tag.Name == "" {
				continue // Пропускаем пустые теги
			}

			// Вставляем или получаем ID тега
			err := s.db.QueryRow(
				ctx,
				`INSERT INTO tags (name) 
                 VALUES ($1)
                 ON CONFLICT (name) DO UPDATE
                 SET name = EXCLUDED.name
                 RETURNING id`,
				tag.Name,
			).Scan(&tag.ID)
			if err != nil {
				return fmt.Errorf("failed to save tag: %w", err)
			}

			// Создаем связь
			_, err = s.db.Exec(
				ctx,
				`INSERT INTO pet_to_tags (pet_id, tag_id) 
                 VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`,
				pet.ID,
				tag.ID,
			)
			if err != nil {
				return fmt.Errorf("failed to link tag: %w", err)
			}
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
         LEFT JOIN pet_categories c ON p.category_id = c.id
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
         FROM pet_tags t
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

func (s *Store) FindPetsByStatus(ctx context.Context, status string) ([]*model.Pet, error) {
	var pets []*model.Pet

	// Запрос основных данных о питомцах с указанным статусом
	rows, err := s.db.Query(
		ctx,
		`SELECT p.id, p.name, p.status, p.photo_urls, p.category_id, c.name
         FROM pets p
         LEFT JOIN pet_categories c ON p.category_id = c.id
         WHERE p.status = $1`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось найти питомцев: %w", err)
	}
	defer rows.Close()

	// Обработка результатов запроса
	for rows.Next() {
		pet := &model.Pet{
			Category: &model.Category{}, // Инициализация категории
			Tags:     []*model.Tag{},    // Инициализация списка тегов
		}

		var categoryID *int64 // Для nullable поля category_id

		// Сканирование данных из строки результата
		if err := rows.Scan(
			&pet.ID,
			&pet.Name,
			&pet.Status,
			pq.Array(&pet.ImageURL), // Используем pq.Array для сканирования массива
			&categoryID,
			&pet.Category.Name,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования данных питомца: %w", err)
		}

		// Установка ID категории (если есть)
		if categoryID != nil {
			pet.Category.ID = *categoryID
		} else {
			pet.Category = nil // Если категория не указана
		}

		// Запрос тегов для текущего питомца
		tagRows, err := s.db.Query(
			ctx,
			`SELECT t.id, t.name
         FROM pet_tags t
         JOIN pet_to_tags pt ON t.id = pt.tag_id
         WHERE pt.pet_id = $1`,
			pet.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("не удалось получить теги: %w", err)
		}
		defer tagRows.Close()

		// Сканирование тегов
		for tagRows.Next() {
			var tag model.Tag
			if err := tagRows.Scan(&tag.ID, &tag.Name); err != nil {
				return nil, fmt.Errorf("ошибка сканирования тега: %w", err)
			}
			pet.Tags = append(pet.Tags, &tag)
		}
		pets = append(pets, pet)
	}
	return pets, nil
}

func (s *Store) UploadImage(ctx context.Context, petID int64, imageURL string) error {
	_, err := s.db.Exec(
		ctx,
		`UPDATE pets 
         SET photo_urls = $1
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
