package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Dkwia/golang-dev-cdek/internal/domain"
	"github.com/Dkwia/golang-dev-cdek/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, fmt.Errorf("email already exists")
		}
		return domain.User{}, err
	}
	return user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (r *Repository) CreateWishlist(ctx context.Context, userID int64, title, description string, eventDate time.Time) (domain.Wishlist, error) {
	var wishlist domain.Wishlist
	token := uuid.NewString()
	err := r.db.QueryRow(ctx, `
		INSERT INTO wishlists (user_id, title, description, event_date, public_token)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, description, event_date, public_token, created_at, updated_at
	`, userID, title, description, eventDate, token).Scan(
		&wishlist.ID,
		&wishlist.UserID,
		&wishlist.Title,
		&wishlist.Description,
		&wishlist.EventDate,
		&wishlist.PublicToken,
		&wishlist.CreatedAt,
		&wishlist.UpdatedAt,
	)
	return wishlist, err
}

func (r *Repository) ListWishlistsByUser(ctx context.Context, userID int64) ([]domain.Wishlist, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, title, description, event_date, public_token, created_at, updated_at
		FROM wishlists
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wishlists []domain.Wishlist
	for rows.Next() {
		var wishlist domain.Wishlist
		if err := rows.Scan(
			&wishlist.ID,
			&wishlist.UserID,
			&wishlist.Title,
			&wishlist.Description,
			&wishlist.EventDate,
			&wishlist.PublicToken,
			&wishlist.CreatedAt,
			&wishlist.UpdatedAt,
		); err != nil {
			return nil, err
		}
		wishlists = append(wishlists, wishlist)
	}
	return wishlists, rows.Err()
}

func (r *Repository) GetWishlistByID(ctx context.Context, userID, wishlistID int64) (domain.Wishlist, error) {
	var wishlist domain.Wishlist
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, title, description, event_date, public_token, created_at, updated_at
		FROM wishlists
		WHERE id = $1 AND user_id = $2
	`, wishlistID, userID).Scan(
		&wishlist.ID,
		&wishlist.UserID,
		&wishlist.Title,
		&wishlist.Description,
		&wishlist.EventDate,
		&wishlist.PublicToken,
		&wishlist.CreatedAt,
		&wishlist.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wishlist{}, ErrNotFound
	}
	if err != nil {
		return domain.Wishlist{}, err
	}

	items, err := r.ListItemsByWishlist(ctx, wishlist.ID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	wishlist.Items = items
	return wishlist, nil
}

func (r *Repository) UpdateWishlist(ctx context.Context, userID, wishlistID int64, title, description string, eventDate time.Time) (domain.Wishlist, error) {
	var wishlist domain.Wishlist
	err := r.db.QueryRow(ctx, `
		UPDATE wishlists
		SET title = $1, description = $2, event_date = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING id, user_id, title, description, event_date, public_token, created_at, updated_at
	`, title, description, eventDate, wishlistID, userID).Scan(
		&wishlist.ID,
		&wishlist.UserID,
		&wishlist.Title,
		&wishlist.Description,
		&wishlist.EventDate,
		&wishlist.PublicToken,
		&wishlist.CreatedAt,
		&wishlist.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wishlist{}, ErrNotFound
	}
	return wishlist, err
}

func (r *Repository) DeleteWishlist(ctx context.Context, userID, wishlistID int64) error {
	commandTag, err := r.db.Exec(ctx, `
		DELETE FROM wishlists
		WHERE id = $1 AND user_id = $2
	`, wishlistID, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateItem(ctx context.Context, userID, wishlistID int64, item domain.Item) (domain.Item, error) {
	if _, err := r.GetWishlistByID(ctx, userID, wishlistID); err != nil {
		return domain.Item{}, err
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO wishlist_items (wishlist_id, title, description, product_url, priority)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, wishlist_id, title, description, product_url, priority, is_reserved, reserved_at, created_at, updated_at
	`, wishlistID, item.Title, item.Description, item.ProductURL, item.Priority).Scan(
		&item.ID,
		&item.WishlistID,
		&item.Title,
		&item.Description,
		&item.ProductURL,
		&item.Priority,
		&item.IsReserved,
		&item.ReservedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (r *Repository) ListItemsByWishlist(ctx context.Context, wishlistID int64) ([]domain.Item, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, wishlist_id, title, description, product_url, priority, is_reserved, reserved_at, created_at, updated_at
		FROM wishlist_items
		WHERE wishlist_id = $1
		ORDER BY priority DESC, created_at DESC
	`, wishlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(
			&item.ID,
			&item.WishlistID,
			&item.Title,
			&item.Description,
			&item.ProductURL,
			&item.Priority,
			&item.IsReserved,
			&item.ReservedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateItem(ctx context.Context, userID, wishlistID, itemID int64, item domain.Item) (domain.Item, error) {
	if _, err := r.GetWishlistByID(ctx, userID, wishlistID); err != nil {
		return domain.Item{}, err
	}

	err := r.db.QueryRow(ctx, `
		UPDATE wishlist_items
		SET title = $1, description = $2, product_url = $3, priority = $4, updated_at = NOW()
		WHERE id = $5 AND wishlist_id = $6
		RETURNING id, wishlist_id, title, description, product_url, priority, is_reserved, reserved_at, created_at, updated_at
	`, item.Title, item.Description, item.ProductURL, item.Priority, itemID, wishlistID).Scan(
		&item.ID,
		&item.WishlistID,
		&item.Title,
		&item.Description,
		&item.ProductURL,
		&item.Priority,
		&item.IsReserved,
		&item.ReservedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, ErrNotFound
	}
	return item, err
}

func (r *Repository) DeleteItem(ctx context.Context, userID, wishlistID, itemID int64) error {
	if _, err := r.GetWishlistByID(ctx, userID, wishlistID); err != nil {
		return err
	}

	commandTag, err := r.db.Exec(ctx, `
		DELETE FROM wishlist_items
		WHERE id = $1 AND wishlist_id = $2
	`, itemID, wishlistID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetPublicWishlist(ctx context.Context, token string) (domain.Wishlist, error) {
	var wishlist domain.Wishlist
	err := r.db.QueryRow(ctx, `
		SELECT id, title, description, event_date, public_token, created_at, updated_at
		FROM wishlists
		WHERE public_token = $1
	`, token).Scan(
		&wishlist.ID,
		&wishlist.Title,
		&wishlist.Description,
		&wishlist.EventDate,
		&wishlist.PublicToken,
		&wishlist.CreatedAt,
		&wishlist.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wishlist{}, ErrNotFound
	}
	if err != nil {
		return domain.Wishlist{}, err
	}

	items, err := r.ListItemsByWishlist(ctx, wishlist.ID)
	if err != nil {
		return domain.Wishlist{}, err
	}
	wishlist.Items = items
	return wishlist, nil
}

func (r *Repository) ReserveItem(ctx context.Context, token string, itemID int64, now time.Time) (domain.Item, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Item{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var wishlistID int64
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM wishlists
		WHERE public_token = $1
	`, token).Scan(&wishlistID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}

	var item domain.Item
	err = tx.QueryRow(ctx, `
		SELECT id, wishlist_id, title, description, product_url, priority, is_reserved, reserved_at, created_at, updated_at
		FROM wishlist_items
		WHERE id = $1 AND wishlist_id = $2
		FOR UPDATE
	`, itemID, wishlistID).Scan(
		&item.ID,
		&item.WishlistID,
		&item.Title,
		&item.Description,
		&item.ProductURL,
		&item.Priority,
		&item.IsReserved,
		&item.ReservedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}

	item, err = service.ReserveItem(item, now)
	if err != nil {
		return domain.Item{}, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE wishlist_items
		SET is_reserved = TRUE, reserved_at = $1, updated_at = $1
		WHERE id = $2
		RETURNING id, wishlist_id, title, description, product_url, priority, is_reserved, reserved_at, created_at, updated_at
	`, now, itemID).Scan(
		&item.ID,
		&item.WishlistID,
		&item.Title,
		&item.Description,
		&item.ProductURL,
		&item.Priority,
		&item.IsReserved,
		&item.ReservedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.Item{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}
