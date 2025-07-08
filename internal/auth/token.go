package auth

import (
	"time"

	"github.com/and161185/loyalty/internal/errs"
	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secretKey []byte
}

func NewTokenManager(secretKey string) *TokenManager {
	return &TokenManager{[]byte(secretKey)}
}

func (tm *TokenManager) GenerateToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // токен на сутки
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secretKey)
}

func (tm *TokenManager) ParseToken(tokenStr string) (int, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errs.ErrInvalidToken
		}
		return tm.secretKey, nil
	})

	if err != nil || !token.Valid {
		return 0, errs.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errs.ErrInvalidToken
	}

	idFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errs.ErrInvalidToken
	}

	return int(idFloat), nil
}
