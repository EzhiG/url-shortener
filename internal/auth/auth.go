package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"crypto/rand"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

type Auth struct {
	secretKey string
}

type ctxKey string

const userIDKey ctxKey = "user_id"
const tokenExp = time.Hour * 5

func New(secretKey string) *Auth {
	return &Auth{secretKey: secretKey}
}

func (auth *Auth) WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func (auth *Auth) UserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return ""
	}

	return userID
}

func (auth *Auth) IsTokenExpiredOnlyError(err error) bool {
	var validationErr *jwt.ValidationError
	return errors.As(err, &validationErr) && validationErr.Errors == jwt.ValidationErrorExpired
}

func (auth *Auth) GenerateUserID() (string, error) {
	key := make([]byte, 16)
	_, err := rand.Read(key)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(key), nil
}

func (auth *Auth) BuildToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(auth.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (auth *Auth) ParseToken(tokenString string) (string, error) {
	claims := &Claims{}
	keyFunc := func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(auth.secretKey), nil
	}
	_, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)

	return claims.UserID, err
}
