package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/a-khrlmv/otus-dz1/internal/service"
)

// RegisterRequest описывает JSON, который
// мы получаем от клиента при регистрации.
//
// Названия полей должны соответствовать OpenAPI.
type RegisterRequest struct {
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Birthdate  string `json:"birthdate"`
	Biography  string `json:"biography"`
	City       string `json:"city"`
	Password   string `json:"password"`
}

// RegisterResponse описывает успешный HTTP-ответ.
//
// Обрати внимание: UserID — string,
// хотя в PostgreSQL идентификатор имеет тип BIGINT.
type RegisterResponse struct {
	UserID string `json:"user_id"`
}

// UserHandler обрабатывает HTTP-запросы,
// связанные с пользователями.
type UserHandler struct {
	service *service.UserService
	logger  *slog.Logger
}

// NewUserHandler создаёт HTTP-обработчик.
func NewUserHandler(
	userService *service.UserService,
	logger *slog.Logger,
) *UserHandler {
	return &UserHandler{
		service: userService,
		logger:  logger,
	}
}

// Register — обработчик POST /user/register.
func (h *UserHandler) Register(
	w http.ResponseWriter,
	r *http.Request,
) {
	// 1. Проверяем Content-Type.
	//
	// mime.ParseMediaType корректно обработает
	// application/json; charset=utf-8.
	mediaType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)

	if err != nil || mediaType != "application/json" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"Content-Type must be application/json",
		)
		return
	}

	// 2. Ограничиваем тело запроса 16 KiB.
	//
	// Для регистрации допускаем больше данных,
	// чем для /login, из-за поля biography.
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	// 3. Декодируем JSON в нашу структуру.
	var req RegisterRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		h.logger.Warn(
			"failed to decode registration request",
			"error", err,
		)

		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid JSON request",
		)
		return
	}

	// 4. Проверяем, что клиент отправил
	// ровно одно JSON-значение.
	var extra any

	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"request must contain one JSON object",
		)
		return
	}

	// 5. Проверяем наличие и длину пароля.
	//
	// bcrypt принимает пароль длиной максимум
	// 72 байта, а не 72 символа.
	if len(req.Password) == 0 || len(req.Password) > 72 {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"password must contain 1 to 72 bytes",
		)
		return
	}

	// 6. Проверяем ограничения VARCHAR,
	// которые установлены в PostgreSQL.
	//
	// utf8.RuneCountInString считает символы Unicode,
	// а не байты. Это важно для русских имён.
	if utf8.RuneCountInString(req.FirstName) > 100 ||
		utf8.RuneCountInString(req.SecondName) > 100 ||
		utf8.RuneCountInString(req.City) > 150 {

		writeJSONError(
			w,
			http.StatusBadRequest,
			"one or more text fields are too long",
		)
		return
	}

	// 7. Разбираем дату рождения.
	//
	// Если поле не передано, сохраняем NULL.
	// Если передано, ожидаем YYYY-MM-DD.
	var birthDate *time.Time

	if req.Birthdate != "" {
		parsedDate, err := time.Parse(
			time.DateOnly,
			req.Birthdate,
		)

		if err != nil {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"birthdate must use YYYY-MM-DD format",
			)
			return
		}

		birthDate = &parsedDate
	}

	// 8. Создаём контекст с таймаутом.
	//
	// Он передаётся в Service и Repository.
	ctx, cancel := context.WithTimeout(
		r.Context(),
		5*time.Second,
	)
	defer cancel()

	// 9. Преобразуем HTTP-данные в данные Service.
	//
	// second_name -> LastName
	// biography   -> Biography (позже interests в БД)
	input := service.RegisterInput{
		FirstName: req.FirstName,
		LastName:  req.SecondName,
		BirthDate: birthDate,
		Biography: req.Biography,
		City:      req.City,
		Password:  req.Password,
	}

	// 10. Выполняем регистрацию.
	userID, err := h.service.Register(ctx, input)

	if err != nil {
		// Записываем подробности в серверные логи,
		// но не раскрываем их HTTP-клиенту.
		h.logger.Error(
			"user registration failed",
			"error", err,
		)

		// Отличаем таймаут от других ошибок.
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.DeadlineExceeded) {

			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"service temporarily unavailable",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	// 11. Формируем успешный ответ.
	//
	// BIGINT преобразуем в string,
	// как требует наша OpenAPI-спецификация.
	response := RegisterResponse{
		UserID: strconv.FormatInt(userID, 10),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error(
			"failed to encode registration response",
			"error", err,
		)
		return
	}

	h.logger.Info(
		"user registered successfully",
		"user_id", userID,
	)
}

//
// Получение анкеты пользователя
//

// GetUserResponse описывает JSON-ответ клиенту.
//
// Имена полей совпадают с принятой структурой User из OpenAPI.
//
// birthdate опускаем, если дата рождения не была указана при регистрации.
type GetUserResponse struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Birthdate  string `json:"birthdate,omitempty"`
	Biography  string `json:"biography"`
	City       string `json:"city"`
}

// GetUser обрабатывает GET /user/get/{id}.
func (h *UserHandler) GetUser(
	w http.ResponseWriter,
	r *http.Request,
) {

	// 1. Получаем ID из параметра URL.
	//
	// Например, для GET /user/get/42
	// r.PathValue("id") вернёт строку "42".
	idParam := r.PathValue("id")

	// 2. Преобразуем строку в int64.
	//
	// Наш PostgreSQL использует BIGINT,
	// поэтому работаем с int64.
	userID, err := strconv.ParseInt(idParam, 10, 64)

	if err != nil || userID <= 0 {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid user id",
		)
		return
	}

	// 3. Создаём контекст с таймаутом.
	//
	// Если запрос к БД выполняется слишком долго, контекст позволит прервать ожидание.
	ctx, cancel := context.WithTimeout(
		r.Context(),
		3*time.Second,
	)
	defer cancel()

	// 4. Вызываем бизнес-логику.
	user, err := h.service.GetUser(ctx, userID)

	if err != nil {

		// Пользователь отсутствует.
		if errors.Is(err, service.ErrUserNotFound) {
			writeJSONError(
				w,
				http.StatusNotFound,
				"user not found",
			)
			return
		}

		// Время обработки истекло.
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(ctx.Err(), context.DeadlineExceeded) {

			h.logger.Warn(
				"get user timed out",
				"user_id", userID,
				"error", err,
			)

			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"service temporarily unavailable",
			)
			return
		}

		// Другая ошибка.
		h.logger.Error(
			"failed to get user",
			"user_id", userID,
			"error", err,
		)

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	// 5. Подготавливаем JSON-ответ.
	//
	// Числовой ID из PostgreSQL переводим в строку согласно OpenAPI-контракту.
	response := GetUserResponse{
		ID:         strconv.FormatInt(user.ID, 10),
		FirstName:  user.FirstName,
		SecondName: user.LastName,
		Biography:  user.Biography,
		City:       user.City,
	}

	// 6. Если дата рождения задана, она преобразуется в формат YYYY-MM-DD.
	if user.BirthDate != nil {
		response.Birthdate = user.BirthDate.Format(
			time.DateOnly,
		)
	}

	// 7. Возвращаем HTTP 200 с JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error(
			"failed to encode user response",
			"user_id", userID,
			"error", err,
		)
		return
	}
}
