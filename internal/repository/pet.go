package repository

import (
	"PetStore/internal/model"
	"context"
	"fmt"
	"strings"

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
	UploadImage(ctx context.Context, petID int64, imageURLs []string) error
	UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error
	DebugPets(ctx context.Context) error
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
	// Начинаем транзакцию
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Создаем или получаем категорию
	var categoryID *int64
	if pet.Category != nil && pet.Category.Name != "" {
		err := tx.QueryRow(ctx,
			`INSERT INTO pet_categories (name) 
             VALUES ($1)
             ON CONFLICT (name) DO UPDATE 
             SET name = EXCLUDED.name 
             RETURNING id`,
			pet.Category.Name,
		).Scan(&categoryID)

		if err != nil {
			return fmt.Errorf("failed to create category: %w", err)
		}
	}

	// 2. Создаем питомца
	var petID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO pets (name, category_id, status, photo_urls)
         VALUES ($1, $2, $3, $4)
         RETURNING id`,
		pet.Name,
		categoryID,
		pet.Status,
		pet.ImageURL,
	).Scan(&petID)

	if err != nil {
		return fmt.Errorf("failed to save pet: %w", err)
	}

	// 3. Создаем теги
	if pet.Tags != nil {
		for _, tag := range pet.Tags {
			if tag.Name == "" {
				continue
			}

			var tagID int64
			err := tx.QueryRow(ctx,
				`INSERT INTO tags (name) 
                 VALUES ($1)
                 ON CONFLICT (name) DO UPDATE 
                 SET name = EXCLUDED.name 
                 RETURNING id`,
				tag.Name,
			).Scan(&tagID)

			if err != nil {
				return fmt.Errorf("failed to create tag: %w", err)
			}

			_, err = tx.Exec(ctx,
				"INSERT INTO pet_to_tags (pet_id, tag_id) VALUES ($1, $2)",
				petID, tagID,
			)
			if err != nil {
				return fmt.Errorf("failed to link tag: %w", err)
			}
		}
	}

	// Завершаем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	pet.ID = petID
	if categoryID != nil {
		pet.Category.ID = *categoryID
	}
	return nil
}
func (s *Store) UpdatePet(ctx context.Context, pet *model.Pet) error {
	// 1. Валидация входных данных
	if pet == nil {
		return fmt.Errorf("pet cannot be nil")
	}
	if pet.ID == 0 {
		return fmt.Errorf("pet ID is required")
	}

	// 2. Проверяем существование питомца
	var exists bool
	err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pets WHERE id = $1)",
		pet.ID,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check pet existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("pet with ID %d not found", pet.ID)
	}

	// 3. Подготавливаем данные для обновления
	updates := make(map[string]interface{})
	updates["name"] = pet.Name

	// Обработка статуса
	if pet.Status != "" {
		switch pet.Status {
		case "available", "pending", "sold":
			updates["status"] = pet.Status
		default:
			updates["status"] = "available"
		}
	}

	// Обработка изображений
	if len(pet.ImageURL) > 0 {
		updates["photo_urls"] = pq.Array(pet.ImageURL)
	}

	// Обработка категории
	if pet.Category != nil && pet.Category.Name != "" {
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
		updates["category_id"] = pet.Category.ID
	}

	// 4. Формируем и выполняем динамический UPDATE запрос
	if len(updates) > 0 {
		query := "UPDATE pets SET "
		params := make([]interface{}, 0)
		i := 1

		for field, value := range updates {
			query += fmt.Sprintf("%s = $%d, ", field, i)
			params = append(params, value)
			i++
		}

		query = strings.TrimSuffix(query, ", ")
		query += fmt.Sprintf(" WHERE id = $%d", i)
		params = append(params, pet.ID)

		result, err := s.db.Exec(ctx, query, params...)
		if err != nil {
			return fmt.Errorf("failed to update pet: %w", err)
		}

		if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
			return fmt.Errorf("no rows were updated")
		}
	}

	// 5. Обновляем теги (если нужно)
	if pet.Tags != nil {
		// Удаляем все текущие теги
		if _, err := s.db.Exec(ctx,
			"DELETE FROM pet_to_tags WHERE pet_id = $1",
			pet.ID,
		); err != nil {
			return fmt.Errorf("failed to clear tags: %w", err)
		}

		// Добавляем новые теги
		for _, tag := range pet.Tags {
			if tag.Name == "" {
				continue
			}

			// Вставляем или обновляем тег
			err := s.db.QueryRow(ctx,
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

			// Связываем тег с питомцем
			if _, err := s.db.Exec(ctx,
				"INSERT INTO pet_to_tags (pet_id, tag_id) VALUES ($1, $2)",
				pet.ID, tag.ID,
			); err != nil {
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

	// Разбиваем строку статусов на массив
	statuses := strings.Split(status, ",")
	for i := range statuses {
		statuses[i] = strings.TrimSpace(statuses[i])
	}

	// Формируем условие WHERE для SQL запроса
	var whereClause string
	var args []interface{}
	if len(statuses) == 1 {
		whereClause = "WHERE p.status::text = $1"
		args = append(args, statuses[0])
	} else {
		placeholders := make([]string, len(statuses))
		for i := range statuses {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, statuses[i])
		}
		whereClause = fmt.Sprintf("WHERE p.status::text IN (%s)", strings.Join(placeholders, ","))
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.name, p.status::text, 
		       ARRAY(SELECT unnest(COALESCE(p.photo_urls, ARRAY[]::text[]))) as photo_urls,
		       p.category_id, c.name
		FROM pets p
		LEFT JOIN pet_categories c ON p.category_id = c.id
		%s
		ORDER BY p.id`, whereClause)

	fmt.Printf("SQL Query: %s\n", query)
	fmt.Printf("Args: %v\n", args)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find pets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		pet := &model.Pet{
			Category: &model.Category{},
			Tags:     []*model.Tag{},
		}

		var (
			categoryID   *int64
			photoURLs    []string
			categoryName *string
		)

		if err := rows.Scan(
			&pet.ID,
			&pet.Name,
			&pet.Status,
			&photoURLs,
			&categoryID,
			&categoryName,
		); err != nil {
			return nil, fmt.Errorf("error scanning pet data: %w", err)
		}

		// Обработка photo_urls
		if photoURLs == nil {
			pet.ImageURL = make([]string, 0)
		} else {
			pet.ImageURL = photoURLs
		}

		// Обработка категории
		if categoryID != nil && categoryName != nil {
			pet.Category = &model.Category{
				ID:   *categoryID,
				Name: *categoryName,
			}
		} else {
			pet.Category = nil
		}

		// Запрос тегов
		tagRows, err := s.db.Query(
			ctx,
			`SELECT t.id, t.name
             FROM pet_tags t
             JOIN pet_to_tags pt ON t.id = pt.tag_id
             WHERE pt.pet_id = $1`,
			pet.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}
		defer tagRows.Close()

		for tagRows.Next() {
			var tag model.Tag
			if err := tagRows.Scan(&tag.ID, &tag.Name); err != nil {
				return nil, fmt.Errorf("error scanning tag: %w", err)
			}
			pet.Tags = append(pet.Tags, &tag)
		}

		pets = append(pets, pet)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pets: %w", err)
	}

	if len(pets) == 0 {
		return nil, fmt.Errorf("no pets found with status: %s", status)
	}

	return pets, nil
}
func (s *Store) UploadImage(ctx context.Context, petID int64, imageURLs []string) error {
	_, err := s.db.Exec(
		ctx,
		`UPDATE pets 
         SET photo_urls = $1
         WHERE id = $2`,
		pq.Array(imageURLs),
		petID,
	)
	if err != nil {
		return fmt.Errorf("failed to update pet images: %w", err)
	}
	return nil
}
func (s *Store) UpdatePetWithForm(ctx context.Context, id int64, name string, status string) error {
	// Валидация статуса
	if status != "" && status != "available" && status != "pending" && status != "sold" {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Формируем запрос в зависимости от переданных параметров
	var query string
	var args []interface{}
	argPos := 1

	if name != "" {
		query = "UPDATE pets SET name = $1"
		args = append(args, name)
		argPos++
	}

	if status != "" {
		if name != "" {
			query += ", status = $" + fmt.Sprint(argPos)
		} else {
			query = "UPDATE pets SET status = $1"
		}
		args = append(args, status)
		argPos++
	}

	if len(args) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query += " WHERE id = $" + fmt.Sprint(argPos)
	args = append(args, id)

	result, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update pet: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("pet with id %d not found", id)
	}
	return nil
}

func (s *Store) DebugPets(ctx context.Context) error {
	query := `SELECT id, name, status, photo_urls::text as photo_urls_text FROM pets`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query pets: %w", err)
	}
	defer rows.Close()

	fmt.Println("\n=== Pets Table Data ===")
	fmt.Printf("%-5s %-20s %-10s %-30s\n", "ID", "Name", "Status", "Photo URLs (raw)")
	fmt.Println("------------------------------------------------------------")

	for rows.Next() {
		var (
			id            int64
			name          string
			status        string
			photoURLsText string
		)

		if err := rows.Scan(&id, &name, &status, &photoURLsText); err != nil {
			return fmt.Errorf("error scanning pet data: %w", err)
		}

		fmt.Printf("%-5d %-20s %-10s %-30s\n", id, name, status, photoURLsText)
	}

	fmt.Println("------------------------------------------------------------")
	return nil
}
