package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-khrlmv/otus-dz1/internal/config"
	"github.com/a-khrlmv/otus-dz1/internal/handler"
	"github.com/a-khrlmv/otus-dz1/internal/middleware"
	"github.com/a-khrlmv/otus-dz1/internal/repository"
	"github.com/a-khrlmv/otus-dz1/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Инициализация логгера
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application stopped with error",
			"error", err,
		)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// 2. Контекст завершения приложения
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// 3. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 4. Создание пула подключений PostgreSQL
	db, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer db.Close()

	// 5. Проверка соединения с PostgreSQL
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	logger.Info("database connection established")

	// ==========================================
	// Инициализация компонентов приложения
	// ==========================================

	// 1. Создаём Repository и передаём ему пул БД.
	authRepo := repository.NewAuthRepository(db)

	// Repository для регистрации пользователей.
	userRepo := repository.NewUserRepository(db)

	// 2. Создаём Service.
	// Передаём Repository и JWT-секрет.
	authService := service.NewAuthService(
		authRepo,
		cfg.JWTSecret,
	)

	// Service содержит логику регистрации.
	userService := service.NewUserService(userRepo)

	// 3. Создаём HTTP Handler.
	// Передаём Service и логгер.
	authHandler := handler.NewAuthHandler(
		authService,
		logger,
	)

	// Handler обрабатывает HTTP-запросы пользователей.
	userHandler := handler.NewUserHandler(userService, logger)

	// ==========================================
	// Регистрация HTTP-маршрутов
	// ==========================================

	mux := http.NewServeMux()

	// Обработчики
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /user/register", userHandler.Register)
	mux.HandleFunc("GET /user/get/{id}", userHandler.GetUser)

	// 7. Создание HTTP-сервера
	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           middleware.Logging(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 8. Запуск сервера в отдельной goroutine
	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("HTTP server starting",
			"port", cfg.HTTPPort,
		)
		serverErrors <- server.ListenAndServe()
	}()

	// 9. Ожидание завершения или ошибки
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}

	case <-ctx.Done():
		logger.Info("shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		<-serverErrors
	}

	logger.Info("application stopped")
	return nil
}
