package domain

import "errors"

// Wallet represents a wallet entity.
// Balance is stored in cents (int64) to avoid float precision issues.
type Wallet struct {
	ID      string `json:"id"`
	Balance int64  `json:"balance"` // in cents
}

var (
	ErrNotFound          = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrSameWallet        = errors.New("source and destination wallets must be different")
)
