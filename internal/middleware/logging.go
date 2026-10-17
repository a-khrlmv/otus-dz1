package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder оборачивает стандартный ResponseWriter,
// чтобы мы могли узнать HTTP-статус ответа.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader перехватывает установку HTTP-статуса.
func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Logging возвращает middleware для логирования, всех HTTP-запросов.
func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Засекаем время начала обработки запроса.
		start := time.Now()

		// По умолчанию HTTP возвращает 200 OK, если обработчик не вызвал WriteHeader.
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Передаём запрос следующему обработчику.
		next.ServeHTTP(recorder, r)

		// После завершения обработки вычисляем длительность.
		duration := time.Since(start)

		// Логируем результат запроса.
		logger.Info(
			"HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", duration.String(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
