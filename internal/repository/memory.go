package repository

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/zgq/wallet/internal/domain"
)

// Repository defines the storage interface for wallets.
type Repository interface {
	Create(ctx context.Context) (*domain.Wallet, error)
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
	Transfer(ctx context.Context, fromID, toID string, amount int64) error
	Deposit(ctx context.Context, id string, amount int64) (*domain.Wallet, error)
}

type memoryRepo struct {
	mu      sync.RWMutex
	wallets map[string]*domain.Wallet
}

// NewMemoryRepo returns a new in-memory Repository.
func NewMemoryRepo() Repository {
	return &memoryRepo{
		wallets: make(map[string]*domain.Wallet),
	}
}

func (r *memoryRepo) Create(_ context.Context) (*domain.Wallet, error) {
	w := &domain.Wallet{
		ID:      uuid.NewString(),
		Balance: 0,
	}
	r.mu.Lock()
	r.wallets[w.ID] = w
	r.mu.Unlock()
	return w, nil
}

func (r *memoryRepo) GetByID(_ context.Context, id string) (*domain.Wallet, error) {
	r.mu.RLock()
	w, ok := r.wallets[id]
	r.mu.RUnlock()
	if !ok {
		return nil, domain.ErrNotFound
	}
	// return a copy to avoid external mutation
	cp := *w
	return &cp, nil
}

func (r *memoryRepo) Deposit(_ context.Context, id string, amount int64) (*domain.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.wallets[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	w.Balance += amount
	cp := *w
	return &cp, nil
}

func (r *memoryRepo) Transfer(_ context.Context, fromID, toID string, amount int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	from, ok := r.wallets[fromID]
	if !ok {
		return domain.ErrNotFound
	}
	to, ok := r.wallets[toID]
	if !ok {
		return domain.ErrNotFound
	}
	if from.Balance < amount {
		return domain.ErrInsufficientFunds
	}
	from.Balance -= amount
	to.Balance += amount
	return nil
}
