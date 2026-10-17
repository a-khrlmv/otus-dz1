package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/a-khrlmv/otus-dz1/internal/repository"
)

// RegisterInput — данные пользователя после
// проверки и преобразования HTTP-запроса.
type RegisterInput struct {
	FirstName string
	LastName  string
	BirthDate *time.Time
	Biography string
	City      string
	Password  string
}

// UserService содержит бизнес-логику
// регистрации пользователей.
type UserService struct {
	repo *repository.UserRepository
}

// NewUserService создаёт сервис регистрации.
func NewUserService(
	repo *repository.UserRepository,
) *UserService {
	return &UserService{
		repo: repo,
	}
}

// Register регистрирует пользователя
// и возвращает его числовой идентификатор.
func (s *UserService) Register(
	ctx context.Context,
	input RegisterInput,
) (int64, error) {

	// 1. Хешируем пароль через bcrypt.
	//
	// Обычный пароль никогда не передаём
	// в Repository и не сохраняем в БД.
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"hash user password: %w",
			err,
		)
	}

	// 2. Генерируем технический уникальный логин.
	//
	// Он необходим из-за ограничения NOT NULL в таблице auth, но не возвращается клиенту.
	login, err := generateInternalLogin()
	if err != nil {
		return 0, fmt.Errorf(
			"generate internal login: %w",
			err,
		)
	}

	// 3. Подготавливаем данные для Repository.
	params := repository.CreateUserParams{
		Login:        login,
		PasswordHash: string(passwordHash),

		FirstName: input.FirstName,
		LastName:  input.LastName,
		BirthDate: input.BirthDate,
		Biography: input.Biography,
		City:      input.City,
	}

	// 4. Сохраняем пользователя и профиль.
	userID, err := s.repo.CreateUser(ctx, params)
	if err != nil {
		return 0, fmt.Errorf(
			"register user: %w",
			err,
		)
	}

	return userID, nil
}

// generateInternalLogin создаёт случайный логин.
func generateInternalLogin() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return "user_" + hex.EncodeToString(randomBytes), nil
}
