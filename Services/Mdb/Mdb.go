package services

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

var db_name string
var postgresURI string

func initEnv() {
	db_name = os.Getenv("POSTGRES_DB")
	if db_name == "" {
		db_name = "hiffi" // Default from docker-compose
	}

	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		postgresUser = "hiffi" // Default from docker-compose
	}

	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	if postgresPassword == "" {
		postgresPassword = "dataofhiffiofsuperlabs" // Default from docker-compose
	}

	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "localhost"
	}

	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}

	// Construct PostgreSQL connection string
	postgresURI = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, db_name)
}

func InitPostgres() {
	initEnv()

	var err error
	DB, err = sql.Open("postgres", postgresURI)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database connection: %v", err))
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := DB.Ping(); err != nil {
		panic(fmt.Sprintf("Failed to ping database: %v", err))
	}

	fmt.Println("PostgreSQL connected!")
}

// RunMigrations runs all SQL migration files in order
func RunMigrations() error {
	migrations := []string{
		"DB/migrations/001_create_users_table.sql",
		"DB/migrations/002_create_deleted_users_table.sql",
		"DB/migrations/003_create_videos_table.sql",
		"DB/migrations/004_create_social_tables.sql",
		"DB/migrations/006_add_password_to_users.sql",
		"DB/migrations/007_add_foreign_key_constraints.sql",
		"DB/migrations/008_create_counters_table.sql",
		"DB/migrations/009_add_bio_email_to_users.sql",
		"DB/migrations/010_add_views_counter.sql",
		"DB/migrations/011_add_videos_created_at_index.sql",
		"DB/migrations/012_add_composite_indexes.sql",
	}

	for _, migrationFile := range migrations {
		migrationSQL, err := os.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", migrationFile, err)
		}

		_, err = DB.Exec(string(migrationSQL))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %v", migrationFile, err)
		}
		fmt.Printf("Migration %s executed successfully\n", migrationFile)
	}

	return nil
}
