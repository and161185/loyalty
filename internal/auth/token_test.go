package auth

import (
	"testing"
	"time"

	"github.com/and161185/loyalty/internal/errs"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func init() {
	SetSecret("testsecret")
}

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken(42)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	userID, err := ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, 42, userID)
}

func TestParseInvalidToken(t *testing.T) {
	_, err := ParseToken("invalid.token.string")
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}

func TestParseTokenWithWrongSignature(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	badTokenStr, _ := token.SignedString([]byte("wrongsecret"))

	_, err := ParseToken(badTokenStr)
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}

func TestParseExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenStr, _ := token.SignedString([]byte("testsecret"))

	_, err := ParseToken(expiredTokenStr)
	require.ErrorIs(t, err, errs.ErrInvalidToken)
}
