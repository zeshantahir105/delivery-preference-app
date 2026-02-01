package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func dsn() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "app")
	pass := getEnv("DB_PASSWORD", "secret")
	name := getEnv("DB_NAME", "orders")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, name)
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Open() (*sql.DB, error) {
	return sql.Open("postgres", dsn())
}

func RunMigrations() error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migratePath := getEnv("MIGRATION_PATH", "file://migrations")
	m, err := migrate.NewWithDatabaseInstance(migratePath, "postgres", driver)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// RunMigrationsDown runs all migrations down (drops schema).
func RunMigrationsDown() error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migratePath := getEnv("MIGRATION_PATH", "file://migrations")
	m, err := migrate.NewWithDatabaseInstance(migratePath, "postgres", driver)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// SeedTestUser ensures user@weel.com exists with password "password" (Go-generated bcrypt).
func SeedTestUser(db *sql.DB) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed: bcrypt failed: %v", err)
		return
	}
	_, err = db.Exec(
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		"user@weel.com", string(hash),
	)
	if err != nil {
		log.Printf("seed: insert test user failed: %v", err)
	}
}
