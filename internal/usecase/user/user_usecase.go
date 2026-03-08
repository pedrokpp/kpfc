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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"kpp.dev/kpfc/internal/domain"
)

type UseCase struct {
	userRepo    domain.UserRepository
	storageRepo domain.StorageRepository
}

func NewUseCase(userRepo domain.UserRepository, storageRepo domain.StorageRepository) *UseCase {
	return &UseCase{
		userRepo:    userRepo,
		storageRepo: storageRepo,
	}
}

func (uc *UseCase) Register(email, password, name string) (*domain.User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}

	if err := validatePassword(password); err != nil {
		return nil, err
	}

	if err := validateName(name); err != nil {
		return nil, err
	}

	existing, err := uc.userRepo.GetByEmail(email)
	if err == nil && existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Password:  string(hashedPassword),
		Provider:  domain.AuthProviderLocal,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (uc *UseCase) Login(email, password string) (*domain.User, error) {
	user, err := uc.userRepo.GetByEmail(email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return user, nil
}

func (uc *UseCase) GetUserByID(id string) (*domain.User, error) {
	return uc.userRepo.GetByID(id)
}

func (uc *UseCase) UpdateUser(id, name string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user.Name = name
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (uc *UseCase) DeleteUser(id string) error {
	return uc.userRepo.Delete(id)
}

func (uc *UseCase) CreateOrGetOAuthUser(email, name, provider string) (*domain.User, error) {
	user, err := uc.userRepo.GetByEmail(email)
	if err == nil && user != nil {
		return user, nil
	}

	user = &domain.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Provider:  domain.AuthProvider(provider),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create oauth user: %w", err)
	}

	return user, nil
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func validateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email cannot be empty: %w", domain.ErrInvalidInput)
	}

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %w", domain.ErrInvalidInput)
	}

	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty: %w", domain.ErrInvalidInput)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters: %w", domain.ErrInvalidInput)
	}

	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("name cannot be empty: %w", domain.ErrInvalidInput)
	}

	return nil
}
