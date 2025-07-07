package auth

import (
	"testing"
	"time"

	"github.com/and161185/loyalty/internal/errs"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken(t *testing.T) {
	tm := TokenManager{secretKey: []byte("testsecret")}
	token, err := tm.GenerateToken(42)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	userID, err := tm.ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, 42, userID)
}

func TestParseInvalidToken(t *testing.T) {
	tm := TokenManager{secretKey: []byte("testsecret")}

	_, err := tm.ParseToken("invalid.token.string")
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}

func TestParseTokenWithWrongSignature(t *testing.T) {
	tm := TokenManager{secretKey: []byte("testsecret")}

	claims := jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	badTokenStr, _ := token.SignedString([]byte("wrongsecret"))

	_, err := tm.ParseToken(badTokenStr)
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}

func TestParseExpiredToken(t *testing.T) {
	tm := TokenManager{secretKey: []byte("testsecret")}

	claims := jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenStr, _ := token.SignedString([]byte("testsecret"))

	_, err := tm.ParseToken(expiredTokenStr)
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}
