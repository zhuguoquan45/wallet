package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/zgq/wallet/internal/domain"
)

type pgRepo struct {
	db *sql.DB
}

// NewPostgresRepo connects to a PostgreSQL database and returns a Repository backed by it.
// The database is created automatically if it does not exist.
func NewPostgresRepo(dsn string) (Repository, error) {
	if err := ensureDatabase(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &pgRepo{db: db}, nil
}

// ensureDatabase connects to the "postgres" maintenance database and creates
// the target database if it does not already exist.
func ensureDatabase(dsn string) error {
	dbName, adminDSN, err := adminDSNFrom(dsn)
	if err != nil {
		return err
	}
	admin, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer admin.Close()

	var exists bool
	if err := admin.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		return fmt.Errorf("check database existence: %w", err)
	}
	if !exists {
		// Database name is controlled by our own config, not user input — safe to interpolate.
		if _, err := admin.Exec(`CREATE DATABASE "` + dbName + `"`); err != nil {
			return fmt.Errorf("create database %q: %w", dbName, err)
		}
	}
	return nil
}

// adminDSNFrom returns the target database name and a DSN pointing to the
// "postgres" maintenance database derived from the original DSN.
func adminDSNFrom(dsn string) (dbName, adminDSN string, err error) {
	u, parseErr := url.Parse(dsn)
	if parseErr != nil {
		return "", "", fmt.Errorf("parse dsn: %w", parseErr)
	}
	dbName = u.Path
	if len(dbName) > 0 && dbName[0] == '/' {
		dbName = dbName[1:]
	}
	if dbName == "" {
		return "", "", fmt.Errorf("no database name in DSN")
	}
	admin := *u
	admin.Path = "/postgres"
	return dbName, admin.String(), nil
}

func (r *pgRepo) Create(ctx context.Context) (*domain.Wallet, error) {
	w := &domain.Wallet{ID: uuid.NewString(), Balance: 0}
	_, err := r.db.ExecContext(ctx, `INSERT INTO wallets (id, balance) VALUES ($1, $2)`, w.ID, w.Balance)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *pgRepo) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	w := &domain.Wallet{}
	err := r.db.QueryRowContext(ctx, `SELECT id, balance FROM wallets WHERE id = $1`, id).
		Scan(&w.ID, &w.Balance)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *pgRepo) Deposit(ctx context.Context, id string, amount int64) (*domain.Wallet, error) {
	w := &domain.Wallet{}
	err := r.db.QueryRowContext(ctx,
		`UPDATE wallets SET balance = balance + $1 WHERE id = $2 RETURNING id, balance`,
		amount, id,
	).Scan(&w.ID, &w.Balance)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *pgRepo) Transfer(ctx context.Context, fromID, toID string, amount int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var fromBal int64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`, fromID).Scan(&fromBal); err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}
	var toBal int64
	if err := tx.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`, toID).Scan(&toBal); err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}
	if fromBal < amount {
		return domain.ErrInsufficientFunds
	}

	if _, err := tx.ExecContext(ctx, `UPDATE wallets SET balance = balance - $1 WHERE id = $2`, amount, fromID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE wallets SET balance = balance + $1 WHERE id = $2`, amount, toID); err != nil {
		return err
	}
	return tx.Commit()
}
