package db

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
)

type Store interface {
	ExecTx(ctx context.Context, fn func(*Queries) error) error
}

type SQLStore struct {
	log *log.Logger
	db  *sql.DB
}

func NewStore(db *sql.DB, log *log.Logger) Store {
	return &SQLStore{
		db:  db,
		log: log,
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
