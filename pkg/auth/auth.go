package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blueambertech/googlesecret"
	"github.com/blueambertech/login-svc-with-gcp/data"
	"github.com/golang-jwt/jwt"
)

var (
	StandardTokenLife = time.Hour * 1
)

// Authorize is a middleware func that checks a http request has a valid JWT before allowing the request to continue
func Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		tokenString, err := getTokenFromHttpHeader(tokenHeader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		err = verifyJWT(r.Context(), data.ProjectID, tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CreateJWT creates a JWT with an additional username, uses a default expiry time of 1 hour
func CreateJWT(ctx context.Context, userName string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": userName,
			"exp":      time.Now().Add(StandardTokenLife).Unix(),
		})

	k, err := getSecretTokenKey(ctx, data.ProjectID)
	if err != nil {
		return "", err
	}
	tokenString, err := token.SignedString(k)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func getTokenFromHttpHeader(header string) (string, error) {
	split := strings.Split(header, "Bearer ")
	if len(split) != 2 {
		return "", errors.New("invalid token format")
	}
	return split[1], nil
}

func verifyJWT(ctx context.Context, projectID, tokenString string) error {
	token, err := jwt.Parse(tokenString, func(_ *jwt.Token) (interface{}, error) {
		return getSecretTokenKey(ctx, projectID)
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return errors.New("token invalid")
	}
	return nil
}

func getSecretTokenKey(ctx context.Context, projectID string) ([]byte, error) {
	secret, err := googlesecret.New(ctx, projectID, "jwt-auth-token-key", "latest")
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}
	return []byte(secret.Value), nil
}
