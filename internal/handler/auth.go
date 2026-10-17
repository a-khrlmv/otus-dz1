package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/a-khrlmv/otus-dz1/internal/repository"
	"github.com/a-khrlmv/otus-dz1/internal/service"
)

// LoginRequest описывает JSON запроса.
//
// JSON: {"id": 42, "password": "secret"}
type LoginRequest struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

// LoginResponse описывает успешный ответ.
type LoginResponse struct {
	Token string `json:"token"`
}

// AuthHandler обрабатывает HTTP-запросы,
// связанные с аутентификацией.
type AuthHandler struct {
	service *service.AuthService
	logger  *slog.Logger
}

// NewAuthHandler создаёт HTTP-обработчик.
func NewAuthHandler(
	authService *service.AuthService,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		service: authService,
		logger:  logger,
	}
}

// Login — обработчик POST /login.
func (h *AuthHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	// 1. Проверяем Content-Type.
	mediaType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)

	if err != nil || mediaType != "application/json" {
		writeJSONError(w, http.StatusBadRequest,
			"Content-Type must be application/json")
		return
	}

	// 2. Ограничиваем размер тела запроса 4 KiB.
	// Это защищает от слишком больших JSON.
	r.Body = http.MaxBytesReader(w, r.Body, 4096)

	// 3. Преобразуем JSON в структуру Go.
	var req LoginRequest

	decoder := json.NewDecoder(r.Body)

	// Запрещаем неизвестные поля в JSON.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		h.logger.Warn(
			"failed to decode login request",
			"error", err,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid JSON request",
		)
		return
	}

	// Запрещаем несколько JSON-объектов
	// в одном HTTP-запросе.
	var extra any

	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest,
			"request must contain one JSON object")
		return
	}

	// 4. Проверяем обязательные поля.
	if req.ID <= 0 {
		writeJSONError(w, http.StatusBadRequest,
			"id must be a positive integer")
		return
	}

	if req.Password == "" || len(req.Password) > 72 {
		writeJSONError(w, http.StatusBadRequest,
			"invalid password length")
		return
	}

	// 5. Устанавливаем таймаут обработки.
	ctx, cancel := context.WithTimeout(
		r.Context(),
		3*time.Second,
	)
	defer cancel()

	// 6. Вызываем бизнес-логику.
	token, err := h.service.Login(
		ctx,
		req.ID,
		req.Password,
	)

	// 7. Обрабатываем возможные ошибки.
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			writeJSONError(w, http.StatusNotFound,
				"user not found")

		case errors.Is(err, service.ErrInvalidPassword):
			writeJSONError(w, http.StatusBadRequest,
				"invalid credentials")

		case errors.Is(err, context.DeadlineExceeded):
			h.logger.Error("login timeout", "error", err)

			writeJSONError(w, http.StatusServiceUnavailable,
				"service temporarily unavailable")

		default:
			// Подробную ошибку логируем на сервере, но не передаём клиенту.
			h.logger.Error("login failed", "error", err)

			writeJSONError(w, http.StatusInternalServerError,
				"internal server error")
		}
		return
	}

	// 8. Авторизация успешна.
	response := LoginResponse{
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error(
			"failed to encode login response",
			"error", err,
		)
	}

	h.logger.Info(
		"user authenticated successfully",
		"user_id", req.ID,
	)
}

// writeJSONError возвращает JSON с описанием ошибки.
func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": message,
		},
	)
}
