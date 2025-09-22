package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Store interface {
	Querier
	ExecTx(ctx context.Context, fn func(*Queries) error) error
}

type SQLStore struct {
	*Queries
	log *log.Logger
	db  *sql.DB
}

func NewStore(db *sql.DB, log *log.Logger) Store {
	return &SQLStore{
		Queries: New(db),
		db:      db,
		log:     log,
	}
}

func (store *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	store.log.InfoWithID(ctx, "[Repository: ExecTx] Called")
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		store.log.ErrorWithID(ctx, "[Repository: ExecTx] Error beginning transaction", err)
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			store.log.ErrorWithID(ctx, "[Repository: ExecTx] Error rolling back transaction", rbErr)
			return fmt.Errorf("tx err: %v, rollback err: %v", err, rbErr)
		}
		store.log.ErrorWithID(ctx, "[Repository: ExecTx] Error executing transaction", err)
		return err
	}

	return tx.Commit()
}

// LogQuery logs SQL queries for debugging
func (store *SQLStore) LogQuery(ctx context.Context, query string, args ...interface{}) {
	start := time.Now()
	store.log.InfoWithID(ctx, fmt.Sprintf("🔍 SQL Query: %s", query), "args", args)

	// Log query execution time
	defer func() {
		duration := time.Since(start)
		store.log.InfoWithID(ctx, fmt.Sprintf("⏱️ Query executed in: %v", duration))
	}()
}
