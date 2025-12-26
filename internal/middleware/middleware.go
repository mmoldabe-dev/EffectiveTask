package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func LogginMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now() // засекаем время старта

			next.ServeHTTP(w, r)

			log.Info("request processed",
				slog.String("method", r.Method), slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Duration("duration", time.Since(t)),
			)
		})
	}
}

func RecoverMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// если все упало — пишем в логи чтоб не пропустить
					log.Error("panic recovred",
						slog.Any("err", err),
						slog.String("url", r.URL.Path))

					http.Error(w, "internal server error", 500)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ставим заголовок для всех ответов
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
