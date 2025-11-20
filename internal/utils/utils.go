package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// context key
type ctxKey string

const CtxUserIDKey ctxKey = "user_id"

// CustomClaims wraps jwt.RegisteredClaims with Email for convenience
type CustomClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// safer subject helper
func (c *CustomClaims) SubjectInt() int64 {
	v, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// Parses TTL such as "15m", "1h", "20s", "30" (minutes)
func parseTTL(ttlStr string) (time.Duration, error) {
	if ttlStr == "" {
		return 15 * time.Minute, nil
	}

	if strings.HasSuffix(ttlStr, "m") ||
		strings.HasSuffix(ttlStr, "h") ||
		strings.HasSuffix(ttlStr, "s") {
		return time.ParseDuration(ttlStr)
	}

	// fallback: minutes
	min, err := strconv.Atoi(ttlStr)
	if err != nil {
		return 0, err
	}
	return time.Duration(min) * time.Minute, nil
}

func GenerateToken(userID int64, email, secret, ttlStr string) (string, int64, error) {
	if secret == "" {
		return "", 0, errors.New("secret not configured")
	}

	dur, err := parseTTL(ttlStr)
	if err != nil {
		return "", 0, err
	}

	now := time.Now()
	expTime := now.Add(dur)

	claims := CustomClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(expTime),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}

	return signed, expTime.Unix(), nil
}

func VerifyToken(tokenStr, secret string) (*CustomClaims, error) {
	if secret == "" {
		return nil, errors.New("secret not configured")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	var claims CustomClaims

	_, err := parser.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 0 {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}
