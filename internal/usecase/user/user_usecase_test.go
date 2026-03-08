// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package user

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"kpp.dev/kpfc/internal/domain"
	"kpp.dev/kpfc/internal/testutil"
)

func TestRegister_EmptyEmail(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	_, err := uc.Register("", "password123", "John")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty email should return ErrInvalidInput")
}

func TestRegister_InvalidEmailFormat(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	invalidEmails := []string{
		"notanemail",
		"@example.com",
		"user@",
		"user @example.com",
	}

	for _, email := range invalidEmails {
		_, err := uc.Register(email, "password123", "John")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInvalidInput), "invalid email format should return ErrInvalidInput")
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	_, err := uc.Register("user@example.com", "", "John")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty password should return ErrInvalidInput")
}

func TestRegister_WeakPassword(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	_, err := uc.Register("user@example.com", "short", "John")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "password < 8 chars should return ErrInvalidInput")
}

func TestRegister_EmptyName(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	_, err := uc.Register("user@example.com", "password123", "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty name should return ErrInvalidInput")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	existingUser := testutil.NewTestUser("user@example.com", "Existing User")
	userRepo.On("GetByEmail", "user@example.com").Return(existingUser, nil)

	_, err := uc.Register("user@example.com", "password123", "New User")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrAlreadyExists), "duplicate email should return ErrAlreadyExists")
	userRepo.AssertExpectations(t)
}

func TestRegister_PasswordHashed(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	userRepo.On("GetByEmail", "user@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.MatchedBy(func(u *domain.User) bool {
		return u.Email == "user@example.com"
	})).Return(nil)

	user, err := uc.Register("user@example.com", "password123", "John")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEqual(t, "password123", user.Password, "password should be hashed")

	// Verificar que a senha foi hasheada com bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123"))
	assert.NoError(t, err, "password should be bcrypt hashed")

	userRepo.AssertExpectations(t)
}

func TestRegister_Success(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	userRepo.On("GetByEmail", "user@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.MatchedBy(func(u *domain.User) bool {
		return u.Email == "user@example.com" && u.Name == "John" && u.Provider == domain.AuthProviderLocal
	})).Return(nil)

	user, err := uc.Register("user@example.com", "password123", "John")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "John", user.Name)
	assert.Equal(t, domain.AuthProviderLocal, user.Provider)
	userRepo.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	user := testutil.NewTestUser("user@example.com", "John")
	user.Password = string(hashedPassword)

	userRepo.On("GetByEmail", "user@example.com").Return(user, nil)

	_, err := uc.Login("user@example.com", "wrongpassword")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidCredentials), "wrong password should return ErrInvalidCredentials")
	userRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	userRepo.On("GetByEmail", "nonexistent@example.com").Return(nil, domain.ErrNotFound)

	_, err := uc.Login("nonexistent@example.com", "password123")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidCredentials), "user not found should return ErrInvalidCredentials (do not leak existence)")
	userRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := testutil.NewTestUser("user@example.com", "John")
	user.Password = string(hashedPassword)

	userRepo.On("GetByEmail", "user@example.com").Return(user, nil)

	result, err := uc.Login("user@example.com", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user@example.com", result.Email)
	userRepo.AssertExpectations(t)
}

func TestCreateOrGetOAuthUser_NewUser(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	userRepo.On("GetByEmail", "oauth@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.MatchedBy(func(u *domain.User) bool {
		return u.Email == "oauth@example.com" && u.Provider == domain.AuthProviderGoogle
	})).Return(nil)

	user, err := uc.CreateOrGetOAuthUser("oauth@example.com", "OAuth User", "google")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "oauth@example.com", user.Email)
	assert.Equal(t, domain.AuthProviderGoogle, user.Provider)
	userRepo.AssertExpectations(t)
}

func TestCreateOrGetOAuthUser_ExistingUser(t *testing.T) {
	userRepo := &testutil.MockUserRepository{}
	storageRepo := &testutil.MockStorageRepository{}
	uc := NewUseCase(userRepo, storageRepo)

	existingUser := testutil.NewTestUser("oauth@example.com", "Existing User")
	existingUser.Provider = domain.AuthProviderGitHub

	userRepo.On("GetByEmail", "oauth@example.com").Return(existingUser, nil)

	user, err := uc.CreateOrGetOAuthUser("oauth@example.com", "OAuth User", "google")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, existingUser.ID, user.ID, "should return existing user")
	assert.Equal(t, domain.AuthProviderGitHub, user.Provider, "provider should remain unchanged")
	userRepo.AssertExpectations(t)
}
