package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/rbconsult-bh/saftaja/internal/store"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already registered")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Service handles user authentication and API key operations.
type Service interface {
	Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	ValidateJWT(tokenStr string) (uuid.UUID, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (uuid.UUID, error)
}

type service struct {
	queries   *store.Queries
	jwtSecret []byte
}

func NewService(queries *store.Queries, jwtSecret []byte) Service {
	return &service{queries: queries, jwtSecret: jwtSecret}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type UserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error) {
	_, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("checking email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, store.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return &UserResponse{ID: user.ID, Email: user.Email, Name: user.Name}, nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error) {
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.issueJWT(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issuing token: %w", err)
	}

	return &TokenResponse{
		Token: token,
		User:  UserResponse{ID: user.ID, Email: user.Email, Name: user.Name},
	}, nil
}

func (s *service) issueJWT(userID uuid.UUID) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *service) GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	return &UserResponse{ID: user.ID, Email: user.Email, Name: user.Name}, nil
}

func (s *service) ValidateJWT(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrUnauthorized
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}

	return userID, nil
}

func (s *service) ValidateAPIKey(ctx context.Context, rawKey string) (uuid.UUID, error) {
	hash := hashKey(rawKey)
	key, err := s.queries.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrUnauthorized
		}
		return uuid.Nil, err
	}

	go func() {
		_ = s.queries.UpdateAPIKeyLastUsed(context.Background(), key.ID)
	}()

	return key.UserID, nil
}

// GenerateAPIKey creates a prefixed random API key. Returns (fullKey, prefix, hash).
func GenerateAPIKey(env string) (fullKey, prefix, keyHash string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return
	}
	random := hex.EncodeToString(b)
	fullKey = fmt.Sprintf("sk_%s_%s", env, random)
	prefix = fullKey[:12]
	keyHash = hashKey(fullKey)
	return
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
