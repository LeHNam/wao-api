package database

import (
	"fmt"
	"github.com/LeHNam/wao-api/config"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresConnection creates a new PostgreSQL connection
// NewPostgresConnection creates a new PostgreSQL connection with connection pooling
func NewPostgresConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.Host
	//dsn := "postgresql://admin:npg_wPvAQJU0lX8E@ep-jolly-block-a1v3g72a-pooler.ap-southeast-1.aws.neon.tech/wao?sslmode=requirefamily=4"
	// Configure connection pooling
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
		WithoutReturning:     false,
	}

	// Open the database connection with GORM
	db, err := gorm.Open(postgres.New(pgConfig), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Error,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
