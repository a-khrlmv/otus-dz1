package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateUserParams содержит все данные, которые Repository должен сохранить в БД.
//
// Repository получает только PasswordHash.
type CreateUserParams struct {
	Login        string
	PasswordHash string

	FirstName string
	LastName  string
	BirthDate *time.Time
	Biography string
	City      string
}

// UserRepository отвечает за SQL-операции с пользователем и его профилем.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository создаёт объект Repository.
//
// Используем уже существующий пул PostgreSQL, который был создан при запуске приложения.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// CreateUser создаёт запись авторизации и профиль.
//
// Обе операции выполняются в одной транзакции.
// Возвращается ID нового пользователя.
func (repo *UserRepository) CreateUser(
	ctx context.Context,
	params CreateUserParams,
) (int64, error) {

	// 1. Начинаем SQL-транзакцию.
	tx, err := repo.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf(
			"begin registration transaction: %w",
			err,
		)
	}

	// 2. Если функция завершится до успешного COMMIT, пытаемся откатить транзакцию.
	//
	// После успешного COMMIT повторный Rollback безопасен: транзакция уже завершена.
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 3. Создаём запись для аутентификации.
	//
	// RETURNING user_id позволяет получить ID, сгенерировный автоматически
	const insertAuth = `
        INSERT INTO public.auth (
            login,
            password_hash
        )
        VALUES ($1, $2)
        RETURNING user_id
    `

	var userID int64

	err = tx.QueryRow(
		ctx,
		insertAuth,
		params.Login,
		params.PasswordHash,
	).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf(
			"insert user credentials: %w",
			err,
		)
	}

	// 4. Создаём профиль с тем же user_id.
	//
	// gender не передаём: применится DEFAULT 'unspecified'
	const insertProfile = `
        INSERT INTO public.profiles (
            user_id,
            first_name,
            last_name,
            birth_date,
            interests,
            city
        )
        VALUES ($1, $2, $3, $4, $5, $6)
    `

	_, err = tx.Exec(
		ctx,
		insertProfile,
		userID,
		params.FirstName,
		params.LastName,
		params.BirthDate,
		params.Biography,
		params.City,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"insert user profile: %w",
			err,
		)
	}

	// 5. Подтверждаем транзакцию.
	//
	// Только после COMMIT записи становятся окончательно сохранёнными.
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(
			"commit registration transaction: %w",
			err,
		)
	}

	return userID, nil
}
