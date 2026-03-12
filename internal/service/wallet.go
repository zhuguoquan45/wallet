package service

import (
	"context"

	"github.com/zgq/wallet/internal/domain"
	"github.com/zgq/wallet/internal/repository"
)

// Service defines the wallet business logic interface.
type Service interface {
	CreateWallet(ctx context.Context) (*domain.Wallet, error)
	GetWallet(ctx context.Context, id string) (*domain.Wallet, error)
	Transfer(ctx context.Context, fromID, toID string, amount int64) error
	Deposit(ctx context.Context, id string, amount int64) (*domain.Wallet, error)
}

type walletService struct {
	repo repository.Repository
}

// New returns a new Service backed by the given repository.
func New(repo repository.Repository) Service {
	return &walletService{repo: repo}
}

func (s *walletService) CreateWallet(ctx context.Context) (*domain.Wallet, error) {
	return s.repo.Create(ctx)
}

func (s *walletService) GetWallet(ctx context.Context, id string) (*domain.Wallet, error) {
	if id == "" {
		return nil, domain.ErrNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *walletService) Deposit(ctx context.Context, id string, amount int64) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	return s.repo.Deposit(ctx, id, amount)
}

func (s *walletService) Transfer(ctx context.Context, fromID, toID string, amount int64) error {
	if amount <= 0 {
		return domain.ErrInvalidAmount
	}
	if fromID == toID {
		return domain.ErrSameWallet
	}
	return s.repo.Transfer(ctx, fromID, toID, amount)
}
