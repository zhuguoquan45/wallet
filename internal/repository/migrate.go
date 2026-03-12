package repository

import "database/sql"

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS wallets (
		id      TEXT PRIMARY KEY,
		balance BIGINT NOT NULL DEFAULT 0
	)`)
	return err
}
