package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// --- Argon2id helpers ---

func TestHashPassword_Format(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=1,p=4$") {
		t.Errorf("unexpected hash format: %s", hash)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("expected 6 parts, got %d: %s", len(parts), hash)
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	hash, err := hashPassword("hunter2")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword("hunter2", hash) {
		t.Error("expected password to verify successfully")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, err := hashPassword("hunter2")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if verifyPassword("wrongpassword", hash) {
		t.Error("expected wrong password to fail verification")
	}
}

func TestVerifyPassword_TamperedHash(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	// Replace the last character to tamper the stored hash
	tampered := hash[:len(hash)-1] + "X"
	if verifyPassword("secret", tampered) {
		t.Error("expected tampered hash to fail verification")
	}
}

func TestVerifyPassword_MalformedEncoding(t *testing.T) {
	if verifyPassword("secret", "notahash") {
		t.Error("expected malformed encoding to fail")
	}
}

// --- JWT ---

func newTestAuthService() *AuthService {
	return &AuthService{jwtSecret: []byte("test-secret")}
}

func TestGenerateToken_Format(t *testing.T) {
	svc := newTestAuthService()
	token, err := svc.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3-part token, got %d parts", len(parts))
	}
}

func TestValidateToken_Valid(t *testing.T) {
	svc := newTestAuthService()
	token, err := svc.GenerateToken(7)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != 7 {
		t.Errorf("userID = %d, want 7", userID)
	}
}

func TestValidateToken_Tampered(t *testing.T) {
	svc := newTestAuthService()
	token, _ := svc.GenerateToken(1)
	parts := strings.Split(token, ".")
	// Flip one byte in the signature
	sig := []byte(parts[2])
	sig[0] ^= 0xFF
	tampered := parts[0] + "." + parts[1] + "." + string(sig)
	if _, err := svc.ValidateToken(tampered); err == nil {
		t.Error("expected tampered token to fail validation")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := &AuthService{jwtSecret: []byte("secret-a")}
	svc2 := &AuthService{jwtSecret: []byte("secret-b")}
	token, _ := svc1.GenerateToken(1)
	if _, err := svc2.ValidateToken(token); err == nil {
		t.Error("expected token signed with different secret to fail")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	svc := newTestAuthService()

	type jwtHeader struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	type jwtClaims struct {
		Sub uint  `json:"sub"`
		Exp int64 `json:"exp"`
		Iat int64 `json:"iat"`
	}

	h, _ := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	c, _ := json.Marshal(jwtClaims{
		Sub: 1,
		Exp: time.Now().Add(-time.Hour).Unix(), // already expired
		Iat: time.Now().Unix(),
	})

	hEnc := base64.RawURLEncoding.EncodeToString(h)
	cEnc := base64.RawURLEncoding.EncodeToString(c)
	expiredToken := hEnc + "." + cEnc + "." + svc.sign(hEnc+"."+cEnc)

	if _, err := svc.ValidateToken(expiredToken); err == nil {
		t.Error("expected expired token to fail validation")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	svc := newTestAuthService()
	if _, err := svc.ValidateToken("not.a.token.with.extra.parts"); err == nil {
		t.Error("expected malformed token to fail")
	}
	if _, err := svc.ValidateToken("onlytwoparts.here"); err == nil {
		t.Error("expected two-part token to fail")
	}
}
