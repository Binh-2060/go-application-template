package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptySecret  = errors.New("jwt: secret must not be empty")
	ErrInvalidToken = errors.New("jwt: invalid token")
	ErrExpiredToken = errors.New("jwt: token expired")
)

// Claims is the payload carried by every token this package issues.
type Claims struct {
	jwt.RegisteredClaims
}

// Manager signs and verifies HS256 tokens with a fixed secret and issuer.
type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

/*
Build a Manager from cfg.

Returns ErrEmptySecret rather than letting a blank JWT_SECRET silently
produce forgeable tokens.
*/
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Secret == "" {
		return nil, ErrEmptySecret
	}

	return &Manager{
		secret: []byte(cfg.Secret),
		issuer: cfg.Issuer,
		ttl:    cfg.TTL,
	}, nil
}

// Sign issues a token for subject, valid for the Manager's configured TTL.
func (m *Manager) Sign(subject string) (string, error) {
	return m.SignWithTTL(subject, m.ttl)
}

// SignWithTTL issues a token for subject with a caller-chosen lifetime,
// letting the same Manager mint both short-lived access tokens and
// longer-lived refresh tokens.
func (m *Manager) SignWithTTL(subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

/*
Parse and validate tokenString, returning its claims.

Rejects tokens signed with anything other than HMAC (mitigates the
alg:"none" / algorithm-confusion class of attack) and tokens whose issuer
doesn't match the Manager's.
*/
func (m *Manager) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
