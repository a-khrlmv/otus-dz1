package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/a-khrlmv/otus-dz1/internal/repository"
)

// ErrInvalidPassword — введён неправильный пароль.
var ErrInvalidPassword = errors.New("invalid password")

// AuthService реализует логику аутентификации.
type AuthService struct {
	repo      *repository.AuthRepository
	jwtSecret []byte
}

// NewAuthService создаёт сервис аутентификации.
func NewAuthService(
	repo *repository.AuthRepository,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Login проверяет пользователя и возвращает JWT.
func (s *AuthService) Login(
	ctx context.Context,
	userID int64,
	password string,
) (string, error) {

	// 1. Получаем хеш пароля из PostgreSQL.
	passwordHash, err := s.repo.GetPasswordHashByID(
		ctx,
		userID,
	)

	if err != nil {
		return "", err
	}

	// 2. Сравниваем введённый пароль с хешем.
	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	if err != nil {
		// Пароль не совпал.
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrInvalidPassword
		}

		// Например, сохранённый хеш повреждён
		// или имеет неправильный формат.
		return "", fmt.Errorf(
			"compare password hash: %w",
			err,
		)
	}

	// 3. Пароль правильный. Создаём JWT.
	now := time.Now().UTC()

	claims := jwt.RegisteredClaims{
		Subject: strconv.FormatInt(userID, 10),
		Issuer:  "social-api",

		IssuedAt: jwt.NewNumericDate(now),

		ExpiresAt: jwt.NewNumericDate(
			now.Add(1 * time.Hour),
		),
	}

	// 4. Указываем алгоритм подписи HS256.
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	// 5. Подписываем токен секретным ключом.
	signedToken, err := token.SignedString(
		s.jwtSecret,
	)

	if err != nil {
		return "", fmt.Errorf(
			"sign JWT: %w",
			err,
		)
	}

	return signedToken, nil
}
