package repository

import (
	"PetStore/internal/model"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StoreRepository interface {
	GetInventory(ctx context.Context) (map[string]int64, error)
	PlaceOrder(ctx context.Context, order *model.Order) error
	GetOrderById(ctx context.Context, id int64) (*model.Order, error)
	DeleteOrder(ctx context.Context, id int64) error
}

type storeRepository struct {
	db *pgxpool.Pool
}

func NewStoreRepository(db *pgxpool.Pool) StoreRepository {
	return &storeRepository{db: db}
}

func (r *storeRepository) PlaceOrder(ctx context.Context, order *model.Order) error {
	// Устанавливаем значение по умолчанию для status, если оно не задано
	if order.Status == "" {
		order.Status = "placed"
	}

	// Проверяем, нет ли уже активных заказов для этого питомца
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM orders 
			WHERE pet_id = $1 
			AND status IN ('placed', 'approved')
		)`,
		order.PetID,
	).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("pet is already in an active order")
	}

	query := `INSERT INTO orders (user_id, pet_id, quantity, ship_date, status, complete) 
              VALUES ($1, $2, $3, $4, $5, $6) 
              RETURNING id, ship_date`

	return r.db.QueryRow(ctx, query,
		order.UserID,
		order.PetID,
		order.Quantity,
		time.Now(),
		order.Status,
		order.Complete,
	).Scan(&order.ID, &order.ShipDate)
}

func (r *storeRepository) GetOrderById(ctx context.Context, id int64) (*model.Order, error) {
	query := `SELECT id, user_id, pet_id, quantity, ship_date, status, complete
              FROM orders WHERE id = $1`

	var order model.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserID, // Новое поле
		&order.PetID,
		&order.Quantity,
		&order.ShipDate,
		&order.Status,
		&order.Complete,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}

func (r *storeRepository) DeleteOrder(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM orders WHERE id = $1`, id)
	return err
}

func (r *storeRepository) GetInventory(ctx context.Context) (map[string]int64, error) {
	query := `SELECT status, COUNT(*) FROM pets GROUP BY status`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inventory := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		inventory[status] = count
	}

	return inventory, nil
}
