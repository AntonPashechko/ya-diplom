package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/AntonPashechko/ya-diplom/internal/logger"
)

func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Получение header c токеном
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == `` {
			logger.Error("authorization header is missing")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		//Токен обычно поставляется в формате `Bearer {token-body}`,проверяем, соответствует ли полученный токен этому требованию
		splitted := strings.Split(tokenHeader, " ")
		if len(splitted) != 2 {
			logger.Error("authorization header bad format {`Bearer {token-body}`}")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		id, err := VerifyToken(splitted[1])
		if err != nil {
			logger.Error("cannot verify jwt: %s", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		//Добавляем id пользователя в Context запроса
		ctx := context.WithValue(r.Context(), "id", id)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
