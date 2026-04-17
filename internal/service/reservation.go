package service

import (
	"errors"
	"time"

	"github.com/Dkwia/golang-dev-cdek/internal/domain"
)

var ErrAlreadyReserved = errors.New("item already reserved")

func ReserveItem(item domain.Item, now time.Time) (domain.Item, error) {
	if item.IsReserved {
		return domain.Item{}, ErrAlreadyReserved
	}

	item.IsReserved = true
	item.ReservedAt = &now
	item.UpdatedAt = now
	return item, nil
}
