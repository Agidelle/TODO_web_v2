package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/agidelle/TODO_web_v2/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (h *TaskHandler) Login(passStored, jwtkey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var password struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&password); err != nil {
			sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, errors.New("ошибка десериализации JSON"), nil))
			return
		}
		hash, err := hashPassword(password.Password)
		if err != nil {
			sendJSONError(w, domain.NewCustomError(http.StatusInternalServerError, errors.New("ошибка создания хэша пароля"), nil))
			return
		}
		if password.Password != passStored {
			sendJSONError(w, domain.NewCustomError(http.StatusUnauthorized, errors.New("не правильный пароль"), nil))
			return
		}
		if err != nil {
			sendJSONError(w, domain.NewCustomError(http.StatusUnauthorized, errors.New("неправильный пароль"), nil))
			return
		}
		token, err := GenerateJWT(jwtkey)
		err = json.NewEncoder(w).Encode(map[string]string{"token": token, "hash": hash})
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}
}

func (h *TaskHandler) JWTMiddleware(pass, secretKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				return
			}
			token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("неправильный метод шифрования token: %v", token.Header["alg"])
				}
				return []byte(secretKey), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "не авторизован.", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GenerateJWT(jwtkey string) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	if jwtkey == "" {
		return "", errors.New("отсутствие jwt-key")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenSign, err := token.SignedString([]byte(jwtkey))
	if err != nil {
		return "", errors.New("ошибка подписи jwt")
	}

	return tokenSign, nil
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Пример функции для сравнения с хэшем хранящимся в БД для аутентификации
// неприменимо в данном случае, так как хранение хэша в токене небезопасно, взято из пета
func checkPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
