package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound — пользователь с таким ID отсутствует.
// Ошибка сервиса, не зависящая от pgx.
var ErrUserNotFound = errors.New("user not found")

// AuthRepository отвечает за SQL-запросы к таблице auth.
type AuthRepository struct {
	db *pgxpool.Pool
}

// NewAuthRepository создаёт объект репозитория.
//
// db — пул подключений, созданный в main.go.
// Новый пул здесь создавать не нужно.
func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		db: db,
	}
}

// GetPasswordHashByID возвращает хеш пароля пользователя по его идентификатору.
func (repo *AuthRepository) GetPasswordHashByID(
	ctx context.Context,
	userID int64,
) (string, error) {

	const query = `
        SELECT password_hash
        FROM public.auth
        WHERE user_id = $1
    `

	var passwordHash string

	// QueryRow выполняет SQL-запрос, а Scan считывает значение в переменную.
	err := repo.db.QueryRow(
		ctx,
		query,
		userID,
	).Scan(&passwordHash)

	if err != nil {
		// pgx.ErrNoRows возникает, если SELECT не вернул ни одной строки.
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotFound
		}

		// Любые другие ошибки передаём выше с дополнительным контекстом.
		return "", fmt.Errorf(
			"get password hash by user id: %w",
			err,
		)
	}

	return passwordHash, nil
}
