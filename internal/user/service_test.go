package user

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type fakeRepository struct {
	saveFunc   func(ctx context.Context, createParams CreateParams) (User, error)
	getFunc    func(ctx context.Context, id int64) (User, error)
	deleteFunc func(ctx context.Context, id int64) error
	getAllFunc func(ctx context.Context) ([]User, error)
}

func (fr *fakeRepository) Save(ctx context.Context, createParams CreateParams) (User, error) {
	return fr.saveFunc(ctx, createParams)
}

func (fr *fakeRepository) Get(ctx context.Context, id int64) (User, error) {
	return fr.getFunc(ctx, id)
}

func (fr *fakeRepository) Delete(ctx context.Context, id int64) error {
	return fr.deleteFunc(ctx, id)
}

func (fr *fakeRepository) GetAll(ctx context.Context) ([]User, error) {
	return fr.getAllFunc(ctx)
}

func TestUserService_Create(t *testing.T) {
	t.Parallel()
	repoErr := errors.New("repository save failed")
	testCases := []struct {
		desc         string
		createParams CreateParams
		saveFunc     func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error)
		wantUser     User
		wantErr      error
	}{
		{
			desc:         "successful",
			createParams: CreateParams{Name: "Alex", Age: 20},
			saveFunc: func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error) {
				t.Helper()

				if createParams.Age != 20 {
					t.Errorf("got age %d, expected 20", createParams.Age)
				}

				if createParams.Name != "Alex" {
					t.Errorf("got name %s, expected Alex", createParams.Name)
				}
				return User{ID: 0, Name: "Alex", Age: 20}, nil
			},
			wantUser: User{ID: 0, Name: "Alex", Age: 20},
			wantErr:  nil,
		},
		{
			desc:         "empty_name",
			createParams: CreateParams{Name: "", Age: 20},
			saveFunc: func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error) {
				t.Helper()

				t.Fatalf("repository.Save should not be called for invalid input")
				return User{}, nil
			},
			wantErr: ErrEmptyUserName,
		},
		{
			desc:         "negative_age",
			createParams: CreateParams{Name: "Alex", Age: -123},
			saveFunc: func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error) {
				t.Helper()

				t.Fatalf("repository.Save should not be called for invalid input")
				return User{}, nil
			},
			wantErr: ErrInvalidUserAge,
		},
		{
			desc:         "zero_age",
			createParams: CreateParams{Name: "Alex", Age: 0},
			saveFunc: func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error) {
				t.Helper()

				t.Fatalf("repository.Save should not be called for invalid input")
				return User{}, nil
			},
			wantErr: ErrInvalidUserAge,
		},
		{
			desc:         "error_propagation",
			createParams: CreateParams{Name: "Alex", Age: 20},
			saveFunc: func(t *testing.T, ctx context.Context, createParams CreateParams) (User, error) {
				t.Helper()

				return User{}, repoErr
			},
			wantErr: repoErr,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeRepo := &fakeRepository{
				saveFunc: func(ctx context.Context, createParams CreateParams) (User, error) {
					return tC.saveFunc(t, ctx, createParams)
				},
			}

			service := NewUserService(fakeRepo)
			user, err := service.Create(context.Background(), tC.createParams)

			if !errors.Is(err, tC.wantErr) {
				checkErrorMismatch(t, tC.wantErr, err)
			}

			if user != tC.wantUser {
				t.Fatalf("got user %v, expcected %v", user, tC.wantUser)
			}
		})
	}
}

func TestUserService_Get(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc      string
		idToGet   int64
		getFunc   func(t *testing.T, ctx context.Context, id int64) (User, error)
		wantUser  User
		assertErr func(t *testing.T, err error)
	}{
		{
			desc:    "propagate_result",
			idToGet: 1,
			getFunc: func(t *testing.T, ctx context.Context, id int64) (User, error) {
				t.Helper()

				if id != 1 {
					t.Errorf("got idToGet %d, expected 1", id)
				}

				return User{ID: 1, Name: "Alex", Age: 20}, nil
			},
			wantUser: User{ID: 1, Name: "Alex", Age: 20},
			assertErr: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}
			},
		},
		{
			desc:    "propagate_error",
			idToGet: 1,
			getFunc: func(t *testing.T, ctx context.Context, id int64) (User, error) {
				t.Helper()

				if id != 1 {
					t.Errorf("got idToGet %d, expected 1", id)
				}

				return User{}, NewUserNotFoundError(1)
			},
			wantUser: User{},
			assertErr: func(t *testing.T, err error) {
				checkIDNotFoundError(t, err, 1)
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeRepo := &fakeRepository{
				getFunc: func(ctx context.Context, id int64) (User, error) {
					return tC.getFunc(t, ctx, id)
				},
			}

			service := NewUserService(fakeRepo)
			user, err := service.Get(context.Background(), tC.idToGet)

			tC.assertErr(t, err)

			if user != tC.wantUser {
				t.Fatalf("got %v, expected %v", user, tC.wantUser)
			}
		})
	}
}

func TestUserService_GetAll(t *testing.T) {
	t.Parallel()
	repoError := errors.New("failed to get users")
	testCases := []struct {
		desc       string
		getAllFunc func(t *testing.T, ctx context.Context) ([]User, error)
		wantUsers  []User
		wantErr    error
	}{
		{
			desc: "successful",
			getAllFunc: func(t *testing.T, ctx context.Context) ([]User, error) {
				t.Helper()

				return []User{
					{ID: 0, Name: "u0", Age: 20},
					{ID: 1, Name: "u1", Age: 21},
					{ID: 2, Name: "u2", Age: 22},
				}, nil
			},
			wantUsers: []User{
				{ID: 0, Name: "u0", Age: 20},
				{ID: 1, Name: "u1", Age: 21},
				{ID: 2, Name: "u2", Age: 22},
			},
			wantErr: nil,
		},
		{
			desc: "error_propagation",
			getAllFunc: func(t *testing.T, ctx context.Context) ([]User, error) {
				t.Helper()

				return []User{}, repoError
			},
			wantUsers: []User{},
			wantErr:   repoError,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeRepo := &fakeRepository{
				getAllFunc: func(ctx context.Context) ([]User, error) {
					return tC.getAllFunc(t, ctx)
				},
			}
			service := NewUserService(fakeRepo)

			users, err := service.GetAll(context.Background())

			if !errors.Is(err, tC.wantErr) {
				checkErrorMismatch(t, tC.wantErr, err)
			}

			if !slices.Equal(users, tC.wantUsers) {
				t.Fatalf("got users %v, expected %v", users, tC.wantUsers)
			}
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc       string
		idToDelete int64
		deleteFunc func(t *testing.T, ctx context.Context, id int64) error
		assertErr  func(t *testing.T, err error)
	}{
		{
			desc:       "successful",
			idToDelete: 1,
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				if id != 1 {
					t.Fatalf("got %d, expected 1", id)
				}

				return nil
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				if err != nil {
					t.Fatalf("got unexpected error %v", err)
				}
			},
		},
		{
			desc:       "propagate_error",
			idToDelete: 1,
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				if id != 1 {
					t.Fatalf("got %d, expected 1", id)
				}

				return NewUserNotFoundError(1)
			},
			assertErr: func(t *testing.T, err error) {
				checkIDNotFoundError(t, err, 1)
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeRepo := &fakeRepository{
				deleteFunc: func(ctx context.Context, id int64) error {
					t.Helper()
					return tC.deleteFunc(t, ctx, id)
				},
			}
			service := NewUserService(fakeRepo)

			err := service.Delete(context.Background(), tC.idToDelete)

			tC.assertErr(t, err)
		})
	}
}

func checkErrorMismatch(t *testing.T, wantErr error, err error) {
	t.Helper()

	if wantErr == nil {
		t.Fatalf("got unexpected error %v", err)
	}
	t.Fatalf("got error %v, expected %v", err, wantErr)
}

func checkIDNotFoundError(t *testing.T, err error, notFoundID int64) {
	t.Helper()

	var notFound *ErrUserNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected error of type *ErrUserNotFound, but got %T (%v)", err, err)
	}

	if notFound.UserID != notFoundID {
		t.Fatalf("got notFound.UserID %d, expected %d", notFound.UserID, notFoundID)
	}
}
