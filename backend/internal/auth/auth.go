package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// MaxPasswordBytes is bcrypt's hard input limit. Anything longer is rejected
// before hashing, otherwise bcrypt errors out and the request turns into a 500.
const MaxPasswordBytes = 72

var ErrInvalidToken = errors.New("invalid or expired session")

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Claims carries the user ID (Subject) plus the session version. The version
// must match users.token_version, so bumping it on the user logs out every
// session issued before the bump.
type Claims struct {
	Version int `json:"ver"`
	jwt.RegisteredClaims
}

func NewToken(secret, userID string, version int, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Version: version,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    "trackete",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken validates the signature and returns the user ID and session
// version. Tokens issued before session versions existed have no "ver" claim and
// yield version 0, which matches the column default.
func ParseToken(secret, token string) (string, int, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("trackete"))
	if err != nil || !parsed.Valid || claims.Subject == "" {
		return "", 0, ErrInvalidToken
	}
	return claims.Subject, claims.Version, nil
}
