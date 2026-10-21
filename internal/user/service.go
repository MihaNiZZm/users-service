package user

import (
	"context"
	"errors"
)

var (
	ErrInvalidUserAge = errors.New("age must be a positive number")
	ErrEmptyUserName  = errors.New("name cannot be empty")
)

type Service struct {
	repository *Repository
}

func NewUserService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, u User) (User, error) {
	if u.Age <= 0 {
		return User{}, ErrInvalidUserAge
	}

	if u.Name == "" {
		return User{}, ErrEmptyUserName
	}

	return s.repository.Save(ctx, u)
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) GetAll(ctx context.Context) ([]User, error) {
	return s.repository.GetAll(ctx)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}
