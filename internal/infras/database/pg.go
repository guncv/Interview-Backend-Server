package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
	"github.com/ngrok/sqlmw"
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBConnections struct {
	GormDB *gorm.DB
	SqlDB  *sql.DB
}

type logger struct{ sqlmw.NullInterceptor }

func (logger) BeforeQuery(ctx context.Context, query string, args ...any) (context.Context, error) {
	log.Printf("🔍 SQLC Query: %s, args=%v", query, args)
	return ctx, nil
}

func ConnectPostgres(cfg *config.Config) *DBConnections {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.DatabaseConfig.Host, cfg.DatabaseConfig.User, cfg.DatabaseConfig.Password, cfg.DatabaseConfig.DbName, cfg.DatabaseConfig.Port, constants.SSLModeDisable, constants.DefaultTimezone)

	sql.Register("pg-logged", sqlmw.Driver(&pq.Driver{}, logger{}))

	sqlDB, err := sql.Open("pg-logged", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to open DB: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to open GORM: %v", err)
	}

	return &DBConnections{
		GormDB: gormDB,
		SqlDB:  sqlDB,
	}
}
