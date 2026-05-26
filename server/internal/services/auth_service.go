package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserDTO, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	LoginWithGoogle(ctx context.Context, idToken string) (*dto.AuthResponse, error)
	RefreshToken(ctx context.Context, rawToken string) (*dto.AuthResponse, error)
	Logout(ctx context.Context, jti string, userID uuid.UUID, expiresAt time.Time, rawRefreshToken string) error
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.MessageResponse, error)
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.MessageResponse, error)
}

type authService struct {
	users      repository.UserRepository
	tokens     repository.TokenRepository
	jwt        TokenService
	cache      pkg.CacheProvider
	mailer     pkg.Mailer
	google     *config.GoogleConfig
	jwtCfg     *config.JWTConfig
	pwdReset   config.PasswordResetConfig
	httpClient *http.Client
}

func NewAuthService(
	users repository.UserRepository,
	tokens repository.TokenRepository,
	jwt TokenService,
	cache pkg.CacheProvider,
	mailer pkg.Mailer,
	googleCfg *config.GoogleConfig,
	jwtCfg *config.JWTConfig,
	pwdResetCfg config.PasswordResetConfig,
) AuthService {
	return &authService{
		users:      users,
		tokens:     tokens,
		jwt:        jwt,
		cache:      cache,
		mailer:     mailer,
		google:     googleCfg,
		jwtCfg:     jwtCfg,
		pwdReset:   pwdResetCfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserDTO, error) {
	email := utils.NormalizeEmail(req.Email)
	fullName := strings.TrimSpace(req.FullName)

	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, apperror.NewDuplicateError("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.NewInternalServerError("failed to check email")
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to hash password")
	}

	user := &model.User{
		FullName:     fullName,
		Email:        email,
		PasswordHash: passwordHash,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, apperror.NewInternalServerError("failed to create user")
	}

	return dto.NewUserDTO(user), nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	email := utils.NormalizeEmail(req.Email)

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewUnauthorizedError("invalid credentials")
		}
		return nil, apperror.NewInternalServerError("failed to find user")
	}

	if user.PasswordHash == "" {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}
	if err := utils.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) LoginWithGoogle(ctx context.Context, idToken string) (*dto.AuthResponse, error) {
	info, err := s.verifyGoogleIDToken(ctx, strings.TrimSpace(idToken))
	if err != nil {
		return nil, apperror.NewUnauthorizedError("invalid Google token")
	}

	if user, err := s.users.FindByGoogleID(ctx, info.Sub); err == nil {
		if err := s.syncGoogleAvatarIfEmpty(ctx, user, info.Picture); err != nil {
			return nil, apperror.NewInternalServerError("failed to update profile")
		}
		return s.issueTokens(ctx, user)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.NewInternalServerError("failed to find user")
	}

	if user, err := s.users.FindByEmail(ctx, info.Email); err == nil {
		if user.GoogleID == nil || *user.GoogleID == "" {
			if err := s.users.UpdateGoogleID(ctx, user.ID, info.Sub); err != nil {
				return nil, apperror.NewInternalServerError("failed to link Google account")
			}
			user.GoogleID = &info.Sub
		}
		if err := s.syncGoogleAvatarIfEmpty(ctx, user, info.Picture); err != nil {
			return nil, apperror.NewInternalServerError("failed to update profile")
		}
		return s.issueTokens(ctx, user)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.NewInternalServerError("failed to find user")
	}

	user := &model.User{
		FullName: defaultGoogleName(info.Name),
		Email:    info.Email,
		GoogleID: &info.Sub,
	}
	if pic := strings.TrimSpace(info.Picture); pic != "" {
		user.AvatarURL = &pic
	}
	if err := s.users.Create(ctx, user); err != nil {
		existing, err := s.users.FindByEmail(ctx, info.Email)
		if err != nil {
			return nil, apperror.NewInternalServerError("failed to find user")
		}
		if existing.GoogleID == nil || *existing.GoogleID == "" {
			_ = s.users.UpdateGoogleID(ctx, existing.ID, info.Sub)
			existing.GoogleID = &info.Sub
		}
		if err := s.syncGoogleAvatarIfEmpty(ctx, existing, info.Picture); err != nil {
			return nil, apperror.NewInternalServerError("failed to update profile")
		}
		return s.issueTokens(ctx, existing)
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) RefreshToken(ctx context.Context, rawToken string) (*dto.AuthResponse, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, apperror.NewUnauthorizedError("invalid refresh token")
	}

	hash := sha256Hex(rawToken)
	stored, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewUnauthorizedError("invalid refresh token")
		}
		return nil, apperror.NewInternalServerError("failed to validate refresh token")
	}

	_ = s.tokens.DeleteByHash(ctx, hash)

	if time.Now().After(stored.ExpiresAt) {
		return nil, apperror.NewUnauthorizedError("refresh token expired")
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewUnauthorizedError("invalid refresh token")
		}
		return nil, apperror.NewInternalServerError("failed to find user")
	}

	return s.issueTokens(ctx, user)
}

func (s *authService) Logout(ctx context.Context, jti string, userID uuid.UUID, expiresAt time.Time, rawRefreshToken string) error {
	if userID == uuid.Nil {
		return apperror.NewBadRequestError("invalid session")
	}
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return apperror.NewBadRequestError("invalid session")
	}
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return apperror.NewBadRequestError("refresh_token is required")
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	if err := s.jwt.BlacklistToken(ctx, jti, ttl); err != nil {
		return apperror.NewInternalServerError("failed to revoke token")
	}

	hash := sha256Hex(rawRefreshToken)
	stored, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return apperror.NewInternalServerError("failed to revoke refresh token")
	}
	if stored.UserID != userID {
		return apperror.NewUnauthorizedError("invalid session")
	}
	if err := s.tokens.DeleteByHash(ctx, hash); err != nil {
		return apperror.NewInternalServerError("failed to revoke refresh token")
	}
	return nil
}

func (s *authService) issueTokens(ctx context.Context, user *model.User) (*dto.AuthResponse, error) {
	access, refresh, err := s.jwt.GenerateTokenPair(ctx, user.ID)
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to generate tokens")
	}

	refreshTTL := s.jwtCfg.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = 168 * time.Hour
	}

	rt := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: sha256Hex(refresh),
		ExpiresAt: time.Now().Add(refreshTTL),
	}
	if err := s.tokens.Create(ctx, rt); err != nil {
		return nil, apperror.NewInternalServerError("failed to store refresh token")
	}

	return &dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

type googleTokenInfo struct {
	Aud     string `json:"aud"`
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (s *authService) verifyGoogleIDToken(ctx context.Context, idToken string) (*googleTokenInfo, error) {
	if idToken == "" {
		return nil, errors.New("empty id_token")
	}

	u := url.URL{Scheme: "https", Host: "oauth2.googleapis.com", Path: "/tokeninfo"}
	q := u.Query()
	q.Set("id_token", idToken)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("google tokeninfo failed")
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	info.Email = utils.NormalizeEmail(info.Email)
	if info.Sub == "" || info.Email == "" {
		return nil, errors.New("missing sub or email")
	}
	if s.google != nil && s.google.ClientID != "" && strings.TrimSpace(info.Aud) != strings.TrimSpace(s.google.ClientID) {
		return nil, errors.New("invalid aud")
	}

	return &info, nil
}

func defaultGoogleName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Google User"
	}
	return name
}

func (s *authService) syncGoogleAvatarIfEmpty(ctx context.Context, user *model.User, picture string) error {
	picture = strings.TrimSpace(picture)
	if picture == "" || user == nil {
		return nil
	}
	if user.AvatarURL != nil && strings.TrimSpace(*user.AvatarURL) != "" {
		return nil
	}
	user.AvatarURL = &picture
	return s.users.Update(ctx, user)
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
