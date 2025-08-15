package database

import (
	"database/sql"
	"fmt"
	"log"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBConnections struct {
	GormDB *gorm.DB
	SqlDB  *sql.DB
}

func ConnectPostgres(cfg *config.Config) *DBConnections {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.DatabaseConfig.Host, cfg.DatabaseConfig.User, cfg.DatabaseConfig.Password, cfg.DatabaseConfig.DbName, cfg.DatabaseConfig.Port, constants.SSLModeDisable, constants.DefaultTimezone)

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database via GORM: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("❌ Failed to get *sql.DB from GORM: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	return &DBConnections{
		GormDB: gormDB,
		SqlDB:  sqlDB,
	}
}
