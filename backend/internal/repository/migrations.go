package repository

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	dirs := []string{"migrations", "./migrations", "/app/migrations"}
	var migrationDir string
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			migrationDir = d
			break
		}
	}
	if migrationDir == "" {
		log.Println("no migrations directory found, skipping")
		return nil
	}

	files, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", name).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}

		log.Printf("applying migration: %s", name)
		if _, err := db.Exec(string(content)); err != nil {
			return err
		}

		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
			return err
		}
	}

	return nil
}
