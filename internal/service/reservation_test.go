package service

import (
	"testing"
	"time"

	"github.com/Dkwia/golang-dev-cdek/internal/domain"
)

func TestReserveItem(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	item, err := ReserveItem(domain.Item{ID: 1}, now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !item.IsReserved {
		t.Fatal("expected item to be reserved")
	}
	if item.ReservedAt == nil || !item.ReservedAt.Equal(now) {
		t.Fatal("expected reserved_at to be set")
	}
}

func TestReserveItemAlreadyReserved(t *testing.T) {
	_, err := ReserveItem(domain.Item{ID: 1, IsReserved: true}, time.Now())
	if err != ErrAlreadyReserved {
		t.Fatalf("expected ErrAlreadyReserved, got %v", err)
	}
}
