package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

const jwtBlacklistPrefix = "jwt:blacklist:"

type TokenService interface {
	GenerateTokenPair(ctx context.Context, userID uuid.UUID) (accessToken, refreshToken string, err error)
	ValidateAccessToken(token string) (userID uuid.UUID, jti string, expiresAt time.Time, err error)
	BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

type tokenService struct {
	cfg   *config.JWTConfig
	cache pkg.CacheProvider
}

func NewTokenService(cfg *config.JWTConfig, cache pkg.CacheProvider) TokenService {
	return &tokenService{cfg: cfg, cache: cache}
}

type accessClaims struct {
	jwt.RegisteredClaims
}

func (s *tokenService) GenerateTokenPair(_ context.Context, userID uuid.UUID) (string, string, error) {
	if s.cfg == nil || s.cfg.AccessSecret == "" {
		return "", "", fmt.Errorf("jwt config is required")
	}

	jti := uuid.New().String()
	now := time.Now()
	accessClaims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	access, err := token.SignedString([]byte(s.cfg.AccessSecret))
	if err != nil {
		return "", "", err
	}

	refresh, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *tokenService) ValidateAccessToken(tokenStr string) (uuid.UUID, string, time.Time, error) {
	if s.cfg == nil || s.cfg.AccessSecret == "" {
		return uuid.Nil, "", time.Time{}, fmt.Errorf("jwt config is required")
	}

	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.AccessSecret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, "", time.Time{}, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	if claims.ID == "" || claims.ExpiresAt == nil {
		return uuid.Nil, "", time.Time{}, errors.New("invalid token claims")
	}

	return userID, claims.ID, claims.ExpiresAt.Time, nil
}

func (s *tokenService) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return s.cache.Set(jwtBlacklistPrefix+jti, "1", ttl)
}

func (s *tokenService) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.cache.IsExist(jwtBlacklistPrefix + jti)
}
