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

func (s *Service) Create(ctx context.Context, ur UserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	if ur.Age <= 0 {
		return User{}, ErrInvalidUserAge
	}

	if ur.Name == "" {
		return User{}, ErrEmptyUserName
	}

	user, err := s.repository.Save(ctx, ur)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	user, err := s.repository.Get(ctx, id)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Service) GetAll(ctx context.Context) ([]User, error) {
	return s.repository.GetAll(ctx)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}
