package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/familycotton/api/internal/model"
	"github.com/familycotton/api/internal/repository"
)

type AuthService struct {
	userRepo   *repository.UserRepository
	tokenRepo  *repository.TokenRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.TokenPair, error) {
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid login or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid login or password")
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, req *model.RefreshRequest) (*model.TokenPair, error) {
	hash := hashToken(req.RefreshToken)

	exists, userID, err := s.tokenRepo.ExistsByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, model.NewAppError(model.ErrUnauthorized, "invalid or expired refresh token")
	}

	if err := s.tokenRepo.DeleteByHash(ctx, hash); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, model.NewAppError(model.ErrUnauthorized, "user not found")
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := hashToken(refreshToken)
	return s.tokenRepo.DeleteByHash(ctx, hash)
}

func (s *AuthService) generateTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"exp":  now.Add(s.accessTTL).Unix(),
		"iat":  now.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshRaw := uuid.New().String()
	refreshHash := hashToken(refreshRaw)
	expiresAt := now.Add(s.refreshTTL)

	if err := s.tokenRepo.Create(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshRaw,
	}, nil
}

func (s *AuthService) ParseAccessToken(tokenStr string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, model.NewAppError(model.ErrUnauthorized, "invalid token signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid token")
	}

	sub, _ := claims.GetSubject()
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", model.NewAppError(model.ErrUnauthorized, "invalid token subject")
	}

	role, _ := claims["role"].(string)
	return userID, role, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
