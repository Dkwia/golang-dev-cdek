package domain

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Wishlist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventDate   time.Time `json:"event_date"`
	PublicToken string    `json:"public_token,omitempty"`
	Items       []Item    `json:"items,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Item struct {
	ID          int64      `json:"id"`
	WishlistID  int64      `json:"wishlist_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ProductURL  string     `json:"product_url"`
	Priority    int        `json:"priority"`
	IsReserved  bool       `json:"is_reserved"`
	ReservedAt  *time.Time `json:"reserved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
