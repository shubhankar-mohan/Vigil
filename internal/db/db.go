package db

import (
	"log"
	"os"
	"path/filepath"

	"vigil/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dbPath string) *gorm.DB {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create db directory %s: %v", dir, err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// Enable WAL mode for better concurrent read performance
	db.Exec("PRAGMA journal_mode=WAL")

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&models.Switch{},
		&models.EvalHistory{},
		&models.AutoDiscoveryRule{},
		&models.SignalOccurrence{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	return db
}
