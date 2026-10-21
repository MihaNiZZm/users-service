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
	user, ok := r.storage[id]
	r.mu.RUnlock()

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
	result := slices.Collect(maps.Values(r.storage))
	r.mu.RUnlock()

	return result, nil
}

func (r *Repository) Save(ctx context.Context, ur UserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	id := int64(r.currentID)
	r.currentID++

	newUser := User{
		ID:   id,
		Name: ur.Name,
		Age:  ur.Age,
	}

	r.mu.Lock()
	r.storage[id] = newUser
	r.mu.Unlock()

	return newUser, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	_, ok := r.storage[id]
	if !ok {
		r.mu.Unlock()
		return NewUserNotFoundError(id)
	}

	delete(r.storage, id)
	r.mu.Unlock()
	return nil
}

func (e *ErrUserNotFound) Error() string {
	return fmt.Sprintf("user %d not found", e.UserID)
}

func NewUserNotFoundError(id int64) error {
	return &ErrUserNotFound{UserID: id}
}
