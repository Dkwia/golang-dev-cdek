package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	secret []byte
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{secret: []byte(secret)}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (s *AuthService) CheckPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (s *AuthService) GenerateToken(userID int64, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", userID),
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *AuthService) ParseToken(token string) (int64, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return 0, ErrInvalidCredentials
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrInvalidCredentials
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	var userID int64
	if _, err := fmt.Sscanf(sub, "%d", &userID); err != nil {
		return 0, ErrInvalidCredentials
	}
	return userID, nil
}
