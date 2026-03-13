package service_test

import (
	"context"
	"testing"

	"github.com/zgq/wallet/internal/domain"
	"github.com/zgq/wallet/internal/repository"
	"github.com/zgq/wallet/internal/service"
)

func newSvc() service.Service {
	return service.New(repository.NewMemoryRepo())
}

func TestCreateWallet(t *testing.T) {
	svc := newSvc()
	w, err := svc.CreateWallet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.ID == "" {
		t.Error("expected non-empty ID")
	}
	if w.Balance != 0 {
		t.Errorf("expected zero balance, got %d", w.Balance)
	}
}

func TestGetWallet(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx)
	got, err := svc.GetWallet(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != w.ID {
		t.Errorf("got ID %q, want %q", got.ID, w.ID)
	}
}

func TestGetWallet_NotFound(t *testing.T) {
	svc := newSvc()
	_, err := svc.GetWallet(context.Background(), "nonexistent")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetWallet_EmptyID(t *testing.T) {
	svc := newSvc()
	_, err := svc.GetWallet(context.Background(), "")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeposit(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	w, _ := svc.CreateWallet(ctx)
	got, err := svc.Deposit(ctx, w.ID, 500)
	if err != nil {
		t.Fatal(err)
	}
	if got.Balance != 500 {
		t.Errorf("expected balance 500, got %d", got.Balance)
	}
}

func TestDeposit_InvalidAmount(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	w, _ := svc.CreateWallet(ctx)

	for _, amt := range []int64{0, -1, -100} {
		_, err := svc.Deposit(ctx, w.ID, amt)
		if err != domain.ErrInvalidAmount {
			t.Errorf("amount=%d: expected ErrInvalidAmount, got %v", amt, err)
		}
	}
}

func TestTransfer(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	a, _ := svc.CreateWallet(ctx)
	b, _ := svc.CreateWallet(ctx)
	svc.Deposit(ctx, a.ID, 1000)

	if err := svc.Transfer(ctx, a.ID, b.ID, 300); err != nil {
		t.Fatal(err)
	}

	wa, _ := svc.GetWallet(ctx, a.ID)
	wb, _ := svc.GetWallet(ctx, b.ID)
	if wa.Balance != 700 {
		t.Errorf("sender balance: want 700, got %d", wa.Balance)
	}
	if wb.Balance != 300 {
		t.Errorf("receiver balance: want 300, got %d", wb.Balance)
	}
}

func TestTransfer_InsufficientFunds(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	a, _ := svc.CreateWallet(ctx)
	b, _ := svc.CreateWallet(ctx)
	svc.Deposit(ctx, a.ID, 100)

	err := svc.Transfer(ctx, a.ID, b.ID, 200)
	if err != domain.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestTransfer_SameWallet(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	a, _ := svc.CreateWallet(ctx)
	svc.Deposit(ctx, a.ID, 1000)

	err := svc.Transfer(ctx, a.ID, a.ID, 100)
	if err != domain.ErrSameWallet {
		t.Errorf("expected ErrSameWallet, got %v", err)
	}
}

func TestTransfer_InvalidAmount(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	a, _ := svc.CreateWallet(ctx)
	b, _ := svc.CreateWallet(ctx)

	err := svc.Transfer(ctx, a.ID, b.ID, 0)
	if err != domain.ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransfer_NotFound(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	a, _ := svc.CreateWallet(ctx)
	svc.Deposit(ctx, a.ID, 1000)

	err := svc.Transfer(ctx, a.ID, "ghost", 100)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
