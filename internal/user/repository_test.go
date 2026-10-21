package user

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
)

func TestUserRepository_ConcurrentSave(t *testing.T) {
	t.Parallel()
	t.Run("no_lost_saves", func(t *testing.T) {
		t.Parallel()
		r := NewUserRepository()

		err := create1000UsersConcurrently(r)
		if err != nil {
			t.Fatalf("concurrent save failed: %v", err)
		}

		users, err := r.GetAll(context.Background())
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		usersSize := len(users)
		if usersSize != 1000 {
			t.Fatalf("got %d created users, but expected 1000", usersSize)
		}
	})

	t.Run("all_IDs_unique", func(t *testing.T) {
		t.Parallel()
		r := NewUserRepository()

		err := create1000UsersConcurrently(r)
		if err != nil {
			t.Fatalf("concurrent save failed: %v", err)
		}

		users, err := r.GetAll(context.Background())
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		uniqueIDs := make(map[int64]struct{})
		for _, u := range users {
			uniqueIDs[u.ID] = struct{}{}
		}

		uniqueIDsSize := len(uniqueIDs)
		if uniqueIDsSize != 1000 {
			t.Fatalf("got %d unique IDs, but expected 1000", uniqueIDsSize)
		}
	})
}

func TestUserRepository_Save(t *testing.T) {
	t.Parallel()
	t.Run("correctly_saves_fields", func(t *testing.T) {
		t.Parallel()
		r := NewUserRepository()
		ctx := context.Background()

		tests := []struct {
			name   string
			params CreateParams
		}{
			{name: "user_1", params: CreateParams{Name: "User1", Age: 19}},
			{name: "user_2", params: CreateParams{Name: "User2", Age: 21}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				user, err := r.Save(ctx, tt.params)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if user.Name != tt.params.Name {
					t.Errorf("Name mismatch: got %q, want %q", user.Name, tt.params.Name)
				}
				if user.Age != tt.params.Age {
					t.Errorf("Age mismatch: got %d, want %d", user.Age, tt.params.Age)
				}
			})
		}
	})

	t.Run("generates_sequential_ids", func(t *testing.T) {
		t.Parallel()
		r := NewUserRepository()
		ctx := context.Background()

		user1, err1 := r.Save(ctx, CreateParams{Name: "User1", Age: 19})
		user2, err2 := r.Save(ctx, CreateParams{Name: "User2", Age: 21})

		if err1 != nil || err2 != nil {
			t.Fatalf("failed to setup ID test: err1=%v, err2=%v", err1, err2)
		}

		if user1.ID != 0 {
			t.Errorf("first user ID mismatch: got %d, want 0", user1.ID)
		}
		if user2.ID != 1 {
			t.Errorf("second user ID mismatch: got %d, want 1", user2.ID)
		}
	})
}

func TestUserRepository_Get(t *testing.T) {
	r := NewUserRepository()
	ctx := context.Background()
	simpleUser := saveUser(t, r, "Alex", 17)

	testCases := []struct {
		desc         string
		id           int64
		expectedUser User
		assertErr    func(t *testing.T, err error)
	}{
		{
			desc:         "simple_user",
			id:           simpleUser.ID,
			expectedUser: simpleUser,
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			desc:         "id_not_found",
			id:           999,
			expectedUser: User{},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				var errUserNotFound *ErrUserNotFound
				if !errors.As(err, &errUserNotFound) {
					t.Fatalf("expected error of type *ErrUserNotFound, but got %T (%v)", err, err)
				}

				expectedID := int64(999)
				if errUserNotFound.UserID != expectedID {
					t.Fatalf("expected incorrect ID to be %d, but got %d", expectedID, errUserNotFound.UserID)
				}
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			user, err := r.Get(ctx, tC.id)

			tC.assertErr(t, err)

			if user != tC.expectedUser {
				t.Errorf("got user %v, expected %v", user, tC.expectedUser)
			}
		})
	}
}

func TestUserRepository_Delete(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc  string
		setup func(t *testing.T, r *Repository) int64
	}{
		{
			desc: "successful",
			setup: func(t *testing.T, r *Repository) int64 {
				t.Helper()

				return saveUser(t, r, "Alex", 17).ID
			},
		},
		{
			desc: "id_not_found",
			setup: func(t *testing.T, r *Repository) int64 {
				t.Helper()

				return 999
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			r := NewUserRepository()
			ctx := context.Background()
			idToDelete := tC.setup(t, r)

			err := r.Delete(ctx, idToDelete)

			switch tC.desc {
			case "successful":
				if err != nil {
					t.Fatalf("unexpected error during delete: %v", err)
				}

				_, getErr := r.Get(ctx, idToDelete)
				var notFound *ErrUserNotFound
				if !errors.As(getErr, &notFound) {
					t.Fatalf("expected error of type *ErrUserNotFound after deletion, but got %T (%v)", getErr, getErr)
				}

			case "id_not_found":
				var notFound *ErrUserNotFound
				if !errors.As(err, &notFound) {
					t.Fatalf("expected error of type *ErrUserNotFound, but got %T (%v)", err, err)
				}

				expectedID := int64(999)
				if notFound.UserID != expectedID {
					t.Fatalf("expected incorrect ID to be %d, but got %d", expectedID, notFound.UserID)
				}
			}
		})
	}
}

func TestUserRepository_GetAll(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc     string
		setup    func(t *testing.T, r *Repository)
		expected []User
	}{
		{
			desc:     "empty_slice",
			setup:    func(t *testing.T, r *Repository) {},
			expected: []User{},
		},
		{
			desc: "one_user",
			setup: func(t *testing.T, r *Repository) {
				saveUser(t, r, "User1", 22)
			},
			expected: []User{
				{
					ID:   0,
					Name: "User1",
					Age:  22,
				},
			},
		},
		{
			desc: "many_users",
			setup: func(t *testing.T, r *Repository) {
				for i := range 8 {
					saveUser(t, r, fmt.Sprintf("User%d", i), 20+i)
				}
			},
			expected: []User{
				{
					ID:   0,
					Name: "User0",
					Age:  20,
				},
				{
					ID:   1,
					Name: "User1",
					Age:  21,
				},
				{
					ID:   2,
					Name: "User2",
					Age:  22,
				},
				{
					ID:   3,
					Name: "User3",
					Age:  23,
				},
				{
					ID:   4,
					Name: "User4",
					Age:  24,
				},
				{
					ID:   5,
					Name: "User5",
					Age:  25,
				},
				{
					ID:   6,
					Name: "User6",
					Age:  26,
				},
				{
					ID:   7,
					Name: "User7",
					Age:  27,
				},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			r := NewUserRepository()
			ctx := context.Background()
			tC.setup(t, r)

			got, err := r.GetAll(ctx)
			if err != nil {
				t.Fatalf("got unexpected error: %v", err)
			}

			sortByID := func(a, b User) int {
				if a.ID < b.ID {
					return -1
				}
				if a.ID > b.ID {
					return 1
				}
				return 0
			}

			slices.SortFunc(got, sortByID)
			slices.SortFunc(tC.expected, sortByID)
			if !slices.Equal(got, tC.expected) {
				t.Fatalf("got %v, expected %v", got, tC.expected)
			}
		})
	}
}

func saveUser(t *testing.T, r *Repository, name string, age int) User {
	t.Helper()

	user, err := r.Save(context.Background(), CreateParams{Name: name, Age: age})
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	return user
}

func create1000UsersConcurrently(r *Repository) error {
	errs := make(chan error, 1000)

	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			_, err := r.Save(
				context.Background(),
				CreateParams{Name: "name", Age: 20},
			)
			if err != nil {
				errs <- err
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		return err
	}

	return nil
}
