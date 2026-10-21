package user

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
)

type ErrUserNotFound struct {
	UserID int64
}

type Repository struct {
	storage   map[int64]User
	currentID int64
	mu        sync.RWMutex
}

func NewUserRepository() *Repository {
	return &Repository{
		storage:   make(map[int64]User),
		currentID: 0,
	}
}

func (r *Repository) Get(ctx context.Context, id int64) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.storage[id]

	if !ok {
		return User{}, NewUserNotFoundError(id)
	}

	return user, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]User, error) {
	if err := ctx.Err(); err != nil {
		return []User{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	result := slices.Collect(maps.Values(r.storage))

	return result, nil
}

func (r *Repository) Save(ctx context.Context, u User) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	id := int64(r.currentID)
	r.currentID++

	newUser := User{
		ID:   id,
		Name: u.Name,
		Age:  u.Age,
	}

	r.storage[id] = newUser

	return newUser, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.storage[id]
	if !ok {
		return NewUserNotFoundError(id)
	}

	delete(r.storage, id)
	return nil
}

func (e *ErrUserNotFound) Error() string {
	return fmt.Sprintf("user %d not found", e.UserID)
}

func NewUserNotFoundError(id int64) error {
	return &ErrUserNotFound{UserID: id}
}
