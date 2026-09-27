package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ahror/royal-move/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct{ UserID, Role, Name string }
type contextKey string

const claimsKey contextKey = "claims"

func Token(secret string, user domain.User) (string, error) {
	claims := jwt.MapClaims{"sub": user.ID, "role": user.Role, "name": user.DisplayName, "iat": time.Now().Unix(), "exp": time.Now().Add(7 * 24 * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
func Middleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("invalid signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c := Claims{UserID: value(claims["sub"]), Role: value(claims["role"]), Name: value(claims["name"])}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
	})
}
func FromContext(ctx context.Context) Claims { value, _ := ctx.Value(claimsKey).(Claims); return value }
func value(v any) string                     { s, _ := v.(string); return s }
