package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AuthService handles registration, login, and JWT issuance.
type AuthService struct {
	users     repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(users repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: []byte(jwtSecret)}
}

// Register creates a new user. Returns ErrEmailTaken if the email is already in use.
func (s *AuthService) Register(email, password, displayName string) (*model.User, error) {
	_, err := s.users.FindByEmail(email)
	if err == nil {
		return nil, ErrEmailTaken
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login verifies credentials, updates the login streak, and returns a JWT plus the updated user.
// Returns ErrInvalidCredentials for unknown email or wrong password.
func (s *AuthService) Login(email, password string) (token string, user *model.User, err error) {
	user, err = s.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if !verifyPassword(password, user.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}

	now := time.Now()
	user.LoginStreak, user.LastLoginDate = CalculateStreak(user.LastLoginDate, user.LoginStreak, now)
	if err = s.users.Update(user); err != nil {
		return "", nil, err
	}

	token, err = s.GenerateToken(user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// GenerateToken creates a signed HS256 JWT for the given user ID, valid for 24 hours.
func (s *AuthService) GenerateToken(userID uint) (string, error) {
	type header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	type claims struct {
		Sub uint  `json:"sub"`
		Exp int64 `json:"exp"`
		Iat int64 `json:"iat"`
	}

	hJSON, err := json.Marshal(header{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	now := time.Now()
	cJSON, err := json.Marshal(claims{
		Sub: userID,
		Exp: now.Add(24 * time.Hour).Unix(),
		Iat: now.Unix(),
	})
	if err != nil {
		return "", err
	}

	h := base64.RawURLEncoding.EncodeToString(hJSON)
	c := base64.RawURLEncoding.EncodeToString(cJSON)
	sig := s.sign(h + "." + c)
	return h + "." + c + "." + sig, nil
}

// ValidateToken parses and verifies a JWT, returning the encoded user ID.
func (s *AuthService) ValidateToken(token string) (uint, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("malformed token")
	}

	expectedSig := s.sign(parts[0] + "." + parts[1])
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedSig)) != 1 {
		return 0, errors.New("invalid token signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, errors.New("malformed token claims")
	}

	var c struct {
		Sub uint  `json:"sub"`
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(claimsJSON, &c); err != nil {
		return 0, errors.New("malformed token claims")
	}
	if time.Now().Unix() > c.Exp {
		return 0, errors.New("token expired")
	}
	return c.Sub, nil
}

func (s *AuthService) sign(input string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// hashPassword hashes password with Argon2id and returns an encoded string.
// Format: $argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=1,p=4$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// verifyPassword checks a plaintext password against an encoded Argon2id hash.
func verifyPassword(password, encoded string) bool {
	// expected: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
