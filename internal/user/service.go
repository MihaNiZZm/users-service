package user

import (
	"context"
	"errors"
)

var (
	ErrInvalidUserAge = errors.New("age must be a positive number")
	ErrEmptyUserName  = errors.New("name cannot be empty")
)

type UserRepository interface {
	Get(context.Context, int64) (User, error)
	GetAll(context.Context) ([]User, error)
	Save(context.Context, CreateParams) (User, error)
	Delete(context.Context, int64) error
}

type Service struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, createParams CreateParams) (User, error) {
	if createParams.Age <= 0 {
		return User{}, ErrInvalidUserAge
	}

	if createParams.Name == "" {
		return User{}, ErrEmptyUserName
	}

	return s.repository.Save(ctx, createParams)
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
