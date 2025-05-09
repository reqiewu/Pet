package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
}

func NewPostgresConnection(ctx context.Context) (*pgxpool.Pool, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}
	cfg := Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Username: os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
	pool, err := pgxpool.New(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Failed to ping database: %w", err)
	}
	if err := ApplyMigrations(cfg.ConnectionString()); err != nil {
		return nil, fmt.Errorf("Failed to apply migrations: %w", err)
	}
	fmt.Println("Connected to database")
	return pool, nil
}

func (c Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.Username, c.Password, c.Host, c.Port, c.DBName,
	)
}

func ApplyMigrations(connString string) error {
	// Проверяем существование папки migrations
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist")
	}

	m, err := migrate.New(
		"file://migrations",
		connString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Проверяем состояние миграций
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// Если миграция в "грязном" состоянии, пытаемся исправить
	if dirty {
		log.Printf("Found dirty migration version %d, attempting to fix...", version)
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
	}

	// Применяем миграции
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	conn, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return err
	}
	defer conn.Close()

	var tableExists bool
	err = conn.QueryRow(context.Background(),
		`SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )`).Scan(&tableExists)

	if !tableExists {
		return fmt.Errorf("users table does not exist after migrations")
	}

	return nil
}
