package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dkwia/golang-dev-cdek/internal/config"
	"github.com/Dkwia/golang-dev-cdek/internal/domain"
	"github.com/Dkwia/golang-dev-cdek/internal/httpx"
	"github.com/Dkwia/golang-dev-cdek/internal/service"
	"github.com/Dkwia/golang-dev-cdek/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const userIDKey contextKey = "userID"

type handler struct {
	auth *service.AuthService
	repo *storage.Repository
}

func NewRouter(cfg config.Config, db *pgxpool.Pool) http.Handler {
	h := &handler{
		auth: service.NewAuthService(cfg.JWTSecret),
		repo: storage.NewRepository(db),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.Handle("GET /api/v1/wishlists", h.authMiddleware(http.HandlerFunc(h.listWishlists)))
	mux.Handle("POST /api/v1/wishlists", h.authMiddleware(http.HandlerFunc(h.createWishlist)))
	mux.Handle("GET /api/v1/wishlists/{id}", h.authMiddleware(http.HandlerFunc(h.getWishlist)))
	mux.Handle("PUT /api/v1/wishlists/{id}", h.authMiddleware(http.HandlerFunc(h.updateWishlist)))
	mux.Handle("DELETE /api/v1/wishlists/{id}", h.authMiddleware(http.HandlerFunc(h.deleteWishlist)))
	mux.Handle("POST /api/v1/wishlists/{id}/items", h.authMiddleware(http.HandlerFunc(h.createItem)))
	mux.Handle("PUT /api/v1/wishlists/{id}/items/{itemID}", h.authMiddleware(http.HandlerFunc(h.updateItem)))
	mux.Handle("DELETE /api/v1/wishlists/{id}/items/{itemID}", h.authMiddleware(http.HandlerFunc(h.deleteItem)))
	mux.HandleFunc("GET /api/v1/public/wishlists/{token}", h.getPublicWishlist)
	mux.HandleFunc("POST /api/v1/public/wishlists/{token}/reserve", h.reserveItem)

	return withJSON(mux)
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (h *handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if token == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		userID, err := h.auth.ParseToken(token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateAuth(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.repo.CreateUser(r.Context(), strings.ToLower(req.Email), hash)
	if err != nil {
		httpx.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	token, err := h.auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"user":  user,
		"token": token,
	})
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateAuth(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.repo.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := h.auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"token": token,
	})
}

type wishlistRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventDate   string `json:"event_date"`
}

func (h *handler) listWishlists(w http.ResponseWriter, r *http.Request) {
	wishlists, err := h.repo.ListWishlistsByUser(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list wishlists")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"wishlists": wishlists})
}

func (h *handler) createWishlist(w http.ResponseWriter, r *http.Request) {
	var req wishlistRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	eventDate, err := validateWishlist(req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	wishlist, err := h.repo.CreateWishlist(r.Context(), userIDFromContext(r.Context()), req.Title, req.Description, eventDate)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create wishlist")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, wishlist)
}

func (h *handler) getWishlist(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}

	wishlist, err := h.repo.GetWishlistByID(r.Context(), userIDFromContext(r.Context()), wishlistID)
	if err != nil {
		writeRepoError(w, err, "wishlist not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, wishlist)
}

func (h *handler) updateWishlist(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}

	var req wishlistRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	eventDate, err := validateWishlist(req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	wishlist, err := h.repo.UpdateWishlist(r.Context(), userIDFromContext(r.Context()), wishlistID, req.Title, req.Description, eventDate)
	if err != nil {
		writeRepoError(w, err, "wishlist not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, wishlist)
}

func (h *handler) deleteWishlist(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}

	if err := h.repo.DeleteWishlist(r.Context(), userIDFromContext(r.Context()), wishlistID); err != nil {
		writeRepoError(w, err, "wishlist not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type itemRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ProductURL  string `json:"product_url"`
	Priority    int    `json:"priority"`
}

func (h *handler) createItem(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}

	var req itemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateItem(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.repo.CreateItem(r.Context(), userIDFromContext(r.Context()), wishlistID, domain.Item{
		Title:       req.Title,
		Description: req.Description,
		ProductURL:  req.ProductURL,
		Priority:    req.Priority,
	})
	if err != nil {
		writeRepoError(w, err, "wishlist not found")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, item)
}

func (h *handler) updateItem(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}
	itemID, err := parseInt64Path(r, "itemID")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req itemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validateItem(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.repo.UpdateItem(r.Context(), userIDFromContext(r.Context()), wishlistID, itemID, domain.Item{
		Title:       req.Title,
		Description: req.Description,
		ProductURL:  req.ProductURL,
		Priority:    req.Priority,
	})
	if err != nil {
		writeRepoError(w, err, "item not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

func (h *handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := parseInt64Path(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid wishlist id")
		return
	}
	itemID, err := parseInt64Path(r, "itemID")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	if err := h.repo.DeleteItem(r.Context(), userIDFromContext(r.Context()), wishlistID, itemID); err != nil {
		writeRepoError(w, err, "item not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *handler) getPublicWishlist(w http.ResponseWriter, r *http.Request) {
	wishlist, err := h.repo.GetPublicWishlist(r.Context(), r.PathValue("token"))
	if err != nil {
		writeRepoError(w, err, "wishlist not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, wishlist)
}

type reserveRequest struct {
	ItemID int64 `json:"item_id"`
}

func (h *handler) reserveItem(w http.ResponseWriter, r *http.Request) {
	var req reserveRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.ItemID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.repo.ReserveItem(r.Context(), r.PathValue("token"), req.ItemID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, service.ErrAlreadyReserved) {
			httpx.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		writeRepoError(w, err, "item not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

func validateAuth(req authRequest) error {
	if !strings.Contains(req.Email, "@") {
		return errors.New("email is invalid")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateWishlist(req wishlistRequest) (time.Time, error) {
	if strings.TrimSpace(req.Title) == "" {
		return time.Time{}, errors.New("title is required")
	}
	eventDate, err := time.Parse("2006-01-02", req.EventDate)
	if err != nil {
		return time.Time{}, errors.New("event_date must be in YYYY-MM-DD format")
	}
	return eventDate, nil
}

func validateItem(req itemRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("title is required")
	}
	if req.Priority < 1 || req.Priority > 5 {
		return errors.New("priority must be between 1 and 5")
	}
	return nil
}

func parseInt64Path(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func userIDFromContext(ctx context.Context) int64 {
	value, _ := ctx.Value(userIDKey).(int64)
	return value
}

func writeRepoError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, fallback)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
