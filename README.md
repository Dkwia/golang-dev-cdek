## Wishlist API

REST API service for user registration, personal wishlist management, and public wishlist sharing by unique link.

### Features

- Registration and login with `email` and `password`
- JWT auth for protected endpoints
- Wishlist CRUD
- Wishlist item CRUD
- Public wishlist access by token
- Gift reservation without authorization
- Automatic PostgreSQL migrations on startup
- Graceful shutdown

### Run

Start the project with:

```bash
docker-compose up --build
```

The API will be available at `http://localhost:8080`.

### Environment

All configuration is stored in `.env.example`. `docker-compose` uses this file directly, so no extra setup step is required before startup.

### Example Requests

#### Register

`POST /api/v1/auth/register`

```json
{
  "email": "user@example.com",
  "password": "strongpass123"
}
```

#### Login

`POST /api/v1/auth/login`

```json
{
  "email": "user@example.com",
  "password": "strongpass123"
}
```

#### Create Wishlist

`POST /api/v1/wishlists`

```json
{
  "title": "New Year",
  "description": "Gifts for the holiday",
  "event_date": "2026-12-31"
}
```

Header:

```text
Authorization: Bearer <jwt>
```

#### Create Wishlist Item

`POST /api/v1/wishlists/{id}/items`

```json
{
  "title": "Mechanical Keyboard",
  "description": "75% layout",
  "product_url": "https://example.com/item",
  "priority": 5
}
```

#### Public Wishlist

`GET /api/v1/public/wishlists/{token}`

#### Reserve Gift

`POST /api/v1/public/wishlists/{token}/reserve`

```json
{
  "item_id": 1
}
```

If the item is already reserved, the API returns `409 Conflict`.

### Endpoints

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/wishlists`
- `POST /api/v1/wishlists`
- `GET /api/v1/wishlists/{id}`
- `PUT /api/v1/wishlists/{id}`
- `DELETE /api/v1/wishlists/{id}`
- `POST /api/v1/wishlists/{id}/items`
- `PUT /api/v1/wishlists/{id}/items/{itemID}`
- `DELETE /api/v1/wishlists/{id}/items/{itemID}`
- `GET /api/v1/public/wishlists/{token}`
- `POST /api/v1/public/wishlists/{token}/reserve`

### Local Checks

```bash
go test ./...
go build ./cmd/api
```
