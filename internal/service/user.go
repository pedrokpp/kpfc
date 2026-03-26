package service

import (
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

// UserService handles user profile operations.
type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

// GetProfile returns the user profile for the given ID.
func (s *UserService) GetProfile(userID uint) (*model.User, error) {
	return s.users.FindByID(userID)
}

// UpdateProfile updates the display name for the given user and returns the updated record.
func (s *UserService) UpdateProfile(userID uint, displayName string) (*model.User, error) {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return nil, err
	}
	user.DisplayName = displayName
	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}
