package repository_test

import (
	"context"
	"sync"
	"testing"

	"github.com/zgq/wallet/internal/domain"
	"github.com/zgq/wallet/internal/repository"
)

func setup(t *testing.T) (repository.Repository, context.Context) {
	t.Helper()
	return repository.NewMemoryRepo(), context.Background()
}

func TestMemory_Create(t *testing.T) {
	repo, ctx := setup(t)
	w, err := repo.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if w.ID == "" || w.Balance != 0 {
		t.Errorf("unexpected wallet: %+v", w)
	}
}

func TestMemory_GetByID(t *testing.T) {
	repo, ctx := setup(t)
	w, _ := repo.Create(ctx)

	got, err := repo.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != w.ID {
		t.Errorf("got %q, want %q", got.ID, w.ID)
	}
}

func TestMemory_GetByID_NotFound(t *testing.T) {
	repo, ctx := setup(t)
	_, err := repo.GetByID(ctx, "missing")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemory_GetByID_ReturnsCopy(t *testing.T) {
	repo, ctx := setup(t)
	w, _ := repo.Create(ctx)

	got, _ := repo.GetByID(ctx, w.ID)
	got.Balance = 9999 // mutate the returned copy

	// original should be unchanged
	fresh, _ := repo.GetByID(ctx, w.ID)
	if fresh.Balance != 0 {
		t.Errorf("mutation leaked into repo: balance=%d", fresh.Balance)
	}
}

func TestMemory_Deposit(t *testing.T) {
	repo, ctx := setup(t)
	w, _ := repo.Create(ctx)

	got, err := repo.Deposit(ctx, w.ID, 500)
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 500 {
		t.Errorf("want 500, got %d", got.Balance)
	}

	// second deposit accumulates
	got, _ = repo.Deposit(ctx, w.ID, 200)
	if got.Balance != 700 {
		t.Errorf("want 700, got %d", got.Balance)
	}
}

func TestMemory_Deposit_NotFound(t *testing.T) {
	repo, ctx := setup(t)
	_, err := repo.Deposit(ctx, "ghost", 100)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemory_Transfer(t *testing.T) {
	repo, ctx := setup(t)
	a, _ := repo.Create(ctx)
	b, _ := repo.Create(ctx)
	repo.Deposit(ctx, a.ID, 1000)

	if err := repo.Transfer(ctx, a.ID, b.ID, 400); err != nil {
		t.Fatal(err)
	}

	wa, _ := repo.GetByID(ctx, a.ID)
	wb, _ := repo.GetByID(ctx, b.ID)
	if wa.Balance != 600 {
		t.Errorf("sender: want 600, got %d", wa.Balance)
	}
	if wb.Balance != 400 {
		t.Errorf("receiver: want 400, got %d", wb.Balance)
	}
}

func TestMemory_Transfer_InsufficientFunds(t *testing.T) {
	repo, ctx := setup(t)
	a, _ := repo.Create(ctx)
	b, _ := repo.Create(ctx)
	repo.Deposit(ctx, a.ID, 50)

	err := repo.Transfer(ctx, a.ID, b.ID, 100)
	if err != domain.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestMemory_Transfer_SrcNotFound(t *testing.T) {
	repo, ctx := setup(t)
	b, _ := repo.Create(ctx)

	err := repo.Transfer(ctx, "ghost", b.ID, 100)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemory_Transfer_DstNotFound(t *testing.T) {
	repo, ctx := setup(t)
	a, _ := repo.Create(ctx)
	repo.Deposit(ctx, a.ID, 500)

	err := repo.Transfer(ctx, a.ID, "ghost", 100)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestMemory_ConcurrentTransfers verifies that concurrent transfers between
// the same pair of wallets never lose or create money (run with -race).
func TestMemory_ConcurrentTransfers(t *testing.T) {
	repo, ctx := setup(t)
	a, _ := repo.Create(ctx)
	b, _ := repo.Create(ctx)
	repo.Deposit(ctx, a.ID, 10_000)
	repo.Deposit(ctx, b.ID, 10_000)

	const goroutines = 50
	const transfers = 20

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < transfers; j++ {
				repo.Transfer(ctx, a.ID, b.ID, 10) //nolint:errcheck
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < transfers; j++ {
				repo.Transfer(ctx, b.ID, a.ID, 10) //nolint:errcheck
			}
		}()
	}
	wg.Wait()

	wa, _ := repo.GetByID(ctx, a.ID)
	wb, _ := repo.GetByID(ctx, b.ID)
	if wa.Balance+wb.Balance != 20_000 {
		t.Errorf("money not conserved: a=%d b=%d total=%d", wa.Balance, wb.Balance, wa.Balance+wb.Balance)
	}
}

// TestMemory_ConcurrentDeposits verifies that concurrent deposits are all applied.
func TestMemory_ConcurrentDeposits(t *testing.T) {
	repo, ctx := setup(t)
	w, _ := repo.Create(ctx)

	const goroutines = 100
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo.Deposit(ctx, w.ID, 1) //nolint:errcheck
		}()
	}
	wg.Wait()

	got, _ := repo.GetByID(ctx, w.ID)
	if got.Balance != goroutines {
		t.Errorf("want %d, got %d", goroutines, got.Balance)
	}
}
