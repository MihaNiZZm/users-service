package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/MihaNiZZm/users-service/internal/user"
)

type fakeService struct {
	createFunc func(ctx context.Context, cp user.CreateParams) (user.User, error)
	getFunc    func(ctx context.Context, id int64) (user.User, error)
	getAllFunc func(ctx context.Context) ([]user.User, error)
	deleteFunc func(ctx context.Context, id int64) error
}

func (f *fakeService) Create(ctx context.Context, cp user.CreateParams) (user.User, error) {
	return f.createFunc(ctx, cp)
}

func (f *fakeService) Delete(ctx context.Context, id int64) error {
	return f.deleteFunc(ctx, id)
}

func (f *fakeService) Get(ctx context.Context, id int64) (user.User, error) {
	return f.getFunc(ctx, id)
}

func (f *fakeService) GetAll(ctx context.Context) ([]user.User, error) {
	return f.getAllFunc(ctx)
}

func TestUserHandler_Get(t *testing.T) {
	t.Parallel()
	internalErr := errors.New("repository failure")
	testCases := []struct {
		desc            string
		getFunc         func(t *testing.T, ctx context.Context, id int64) (user.User, error)
		reqTarget       string
		respStatusCode  int
		respContentType string
		wantUser        user.User
		wantErr         AppError
	}{
		{
			desc: "existing_user",
			getFunc: func(t *testing.T, ctx context.Context, id int64) (user.User, error) {
				t.Helper()

				if id != 1 {
					t.Fatalf("got id %d, expected 1", id)
				}
				return user.User{ID: 1, Name: "Alex", Age: 20}, nil
			},
			reqTarget:       "/users/1",
			respStatusCode:  http.StatusOK,
			respContentType: "application/json",
			wantUser:        user.User{ID: 1, Name: "Alex", Age: 20},
			wantErr:         AppError{},
		},
		{
			desc: "invalid_id",
			getFunc: func(t *testing.T, ctx context.Context, id int64) (user.User, error) {
				t.Helper()

				t.Fatalf("unexpected to call service.Get, but called")
				return user.User{}, nil
			},
			reqTarget:       "/users/number",
			respStatusCode:  http.StatusBadRequest,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "bad_request",
				Message: "id must be a number",
			},
		},
		{
			desc: "user_not_found",
			getFunc: func(t *testing.T, ctx context.Context, id int64) (user.User, error) {
				t.Helper()

				return user.User{}, user.NewUserNotFoundError(1)
			},
			reqTarget:       "/users/1",
			respStatusCode:  http.StatusNotFound,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "not_found",
				Message: user.NewUserNotFoundError(1).Error(),
			},
		},
		{
			desc: "internal_error",
			getFunc: func(t *testing.T, ctx context.Context, id int64) (user.User, error) {
				t.Helper()

				return user.User{}, internalErr
			},
			reqTarget:       "/users/1",
			respStatusCode:  http.StatusInternalServerError,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "internal_error",
				Message: "something went wrong",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeService := &fakeService{
				getFunc: func(ctx context.Context, id int64) (user.User, error) {
					t.Helper()

					return tC.getFunc(t, ctx, id)
				},
			}
			resp := serveRequest(t, fakeService, http.MethodGet, tC.reqTarget, nil)
			defer resp.Body.Close()

			assertStatus(t, resp, tC.respStatusCode)

			if resp.Header.Get("Content-Type") != tC.respContentType {
				t.Errorf("got %v, expected %v", resp.Header.Get("Content-Type"), tC.respContentType)
			}

			if resp.StatusCode == http.StatusOK {
				got := decodeJSON[user.User](t, resp.Body)
				if got != tC.wantUser {
					t.Fatalf("got user %v, expected %v", got, tC.wantUser)
				}
			} else {
				gotErr := decodeJSON[AppError](t, resp.Body)
				if gotErr != tC.wantErr {
					t.Fatalf("got AppError %v, expected %v", gotErr, tC.wantErr)
				}
			}
		})
	}
}

func TestUserHandler_List(t *testing.T) {
	t.Parallel()
	internalErr := errors.New("serviceErr")
	testCases := []struct {
		desc            string
		getAllFunc      func(t *testing.T, ctx context.Context) ([]user.User, error)
		respStatusCode  int
		respContentType string
		wantUsers       []user.User
		wantErr         AppError
	}{
		{
			desc: "empty_users",
			getAllFunc: func(t *testing.T, ctx context.Context) ([]user.User, error) {
				t.Helper()

				return []user.User{}, nil
			},
			respStatusCode:  http.StatusOK,
			respContentType: "application/json",
			wantUsers:       []user.User{},
			wantErr:         AppError{},
		},
		{
			desc: "existing_users",
			getAllFunc: func(t *testing.T, ctx context.Context) ([]user.User, error) {
				t.Helper()

				return []user.User{
					{
						ID:   1,
						Name: "Alex",
						Age:  20,
					},
					{
						ID:   2,
						Name: "Bob",
						Age:  21,
					},
					{
						ID:   3,
						Name: "Mark",
						Age:  22,
					},
				}, nil
			},
			respStatusCode:  http.StatusOK,
			respContentType: "application/json",
			wantUsers: []user.User{
				{
					ID:   1,
					Name: "Alex",
					Age:  20,
				},
				{
					ID:   2,
					Name: "Bob",
					Age:  21,
				},
				{
					ID:   3,
					Name: "Mark",
					Age:  22,
				},
			},
			wantErr: AppError{},
		},
		{
			desc: "error_propagation",
			getAllFunc: func(t *testing.T, ctx context.Context) ([]user.User, error) {
				t.Helper()

				return []user.User{}, internalErr
			},
			respStatusCode:  http.StatusInternalServerError,
			respContentType: "application/json",
			wantUsers:       []user.User{},
			wantErr: AppError{
				Error:   "internal_error",
				Message: "something went wrong",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeService := &fakeService{
				getAllFunc: func(ctx context.Context) ([]user.User, error) {
					t.Helper()

					return tC.getAllFunc(t, ctx)
				},
			}
			resp := serveRequest(t, fakeService, http.MethodGet, "/users", nil)
			defer resp.Body.Close()

			assertStatus(t, resp, tC.respStatusCode)

			if resp.Header.Get("Content-Type") != tC.respContentType {
				t.Errorf("got %v, expected %v", resp.Header.Get("Content-Type"), tC.respContentType)
			}

			if resp.StatusCode == http.StatusOK {
				got := decodeJSON[[]user.User](t, resp.Body)
				if !slices.Equal(got, tC.wantUsers) {
					t.Fatalf("got users %v, expected %v", got, tC.wantUsers)
				}
			} else {
				gotErr := decodeJSON[AppError](t, resp.Body)
				if gotErr != tC.wantErr {
					t.Fatalf("got AppError %v, expected %v", gotErr, tC.wantErr)
				}
			}
		})
	}
}

func TestUserHandler_Create(t *testing.T) {
	t.Parallel()
	internalErr := errors.New("serviceErr")
	testCases := []struct {
		desc            string
		reqBodyString   string
		createFunc      func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error)
		respStatusCode  int
		respContentType string
		wantUser        user.User
		wantErr         AppError
	}{
		{
			desc: "successful",
			reqBodyString: `{
				"name": "Alex", 
				"age": 20
			}`,
			createFunc: func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error) {
				t.Helper()

				expectedCreateParams := user.CreateParams{Name: "Alex", Age: 20}
				if createParams != expectedCreateParams {
					t.Fatalf("got %v, expected %v", createParams, expectedCreateParams)
				}

				return user.User{ID: 1, Name: "Alex", Age: 20}, nil
			},
			respStatusCode:  http.StatusCreated,
			respContentType: "application/json",
			wantUser:        user.User{ID: 1, Name: "Alex", Age: 20},
			wantErr:         AppError{},
		},
		{
			desc:          "invalid_request_body",
			reqBodyString: `{gedagedigedagedaoo}`,
			createFunc: func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error) {
				t.Helper()

				t.Fatal("unexpected call: service.Create")

				return user.User{}, nil
			},
			respStatusCode:  http.StatusBadRequest,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "invalid_request_body",
				Message: `request body must be a json with format { "name": "string", "age": 20 }`,
			},
		},
		{
			desc: "invalid_age",
			reqBodyString: `{
				"name": "Alex", 
				"age": -18
			}`,
			createFunc: func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error) {
				t.Helper()

				expectedCreateParams := user.CreateParams{Name: "Alex", Age: -18}
				if createParams != expectedCreateParams {
					t.Fatalf("got %v, expected %v", createParams, expectedCreateParams)
				}

				return user.User{}, user.ErrInvalidUserAge
			},
			respStatusCode:  http.StatusBadRequest,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "invalid_age",
				Message: user.ErrInvalidUserAge.Error(),
			},
		},
		{
			desc: "invalid_name",
			reqBodyString: `{
				"name": "", 
				"age": 18
			}`,
			createFunc: func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error) {
				t.Helper()

				expectedCreateParams := user.CreateParams{Name: "", Age: 18}
				if createParams != expectedCreateParams {
					t.Fatalf("got %v, expected %v", createParams, expectedCreateParams)
				}

				return user.User{}, user.ErrEmptyUserName
			},
			respStatusCode:  http.StatusBadRequest,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "invalid_name",
				Message: user.ErrEmptyUserName.Error(),
			},
		},
		{
			desc: "internal_error",
			reqBodyString: `{
				"name": "Alex", 
				"age": 18
			}`,
			createFunc: func(t *testing.T, ctx context.Context, createParams user.CreateParams) (user.User, error) {
				t.Helper()

				expectedCreateParams := user.CreateParams{Name: "Alex", Age: 18}
				if createParams != expectedCreateParams {
					t.Fatalf("got %v, expected %v", createParams, expectedCreateParams)
				}

				return user.User{}, internalErr
			},
			respStatusCode:  http.StatusInternalServerError,
			respContentType: "application/json",
			wantUser:        user.User{},
			wantErr: AppError{
				Error:   "internal_error",
				Message: "something went wrong",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeService := &fakeService{
				createFunc: func(ctx context.Context, createParams user.CreateParams) (user.User, error) {
					t.Helper()

					return tC.createFunc(t, ctx, createParams)
				},
			}
			resp := serveRequest(t, fakeService, http.MethodPost, "/users", strings.NewReader(tC.reqBodyString))
			defer resp.Body.Close()

			assertStatus(t, resp, tC.respStatusCode)

			if resp.Header.Get("Content-Type") != tC.respContentType {
				t.Errorf("got %v, expected %v", resp.Header.Get("Content-Type"), tC.respContentType)
			}

			if resp.StatusCode == http.StatusCreated {
				got := decodeJSON[user.User](t, resp.Body)
				if got != tC.wantUser {
					t.Fatalf("got user %v, expected %v", got, tC.wantUser)
				}
			} else {
				gotErr := decodeJSON[AppError](t, resp.Body)
				if gotErr != tC.wantErr {
					t.Fatalf("got AppError %v, expected %v", gotErr, tC.wantErr)
				}
			}
		})
	}
}

func TestUserHandler_Delete(t *testing.T) {
	t.Parallel()
	internalErr := errors.New("serviceErr")
	testCases := []struct {
		desc           string
		reqTarget      string
		deleteFunc     func(t *testing.T, ctx context.Context, id int64) error
		respStatusCode int
		wantErr        AppError
	}{
		{
			desc:      "successful",
			reqTarget: "/users/1",
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				if id != 1 {
					t.Fatalf("got %v, expected 1", id)
				}

				return nil
			},
			respStatusCode: http.StatusNoContent,
			wantErr:        AppError{},
		},
		{
			desc:      "not_found",
			reqTarget: "/users/1",
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				if id != 1 {
					t.Fatalf("got %v, expected 1", id)
				}

				return user.NewUserNotFoundError(1)
			},
			respStatusCode: http.StatusNotFound,
			wantErr: AppError{
				Error:   "not_found",
				Message: user.NewUserNotFoundError(1).Error(),
			},
		},
		{
			desc:      "invalid_id",
			reqTarget: "/users/gedagedigedagedaoo",
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				t.Fatal("unexpected call of service.Delete")

				return nil
			},
			respStatusCode: http.StatusBadRequest,
			wantErr: AppError{
				Error:   "bad_request",
				Message: "id must be a number",
			},
		},
		{
			desc:      "internal_error",
			reqTarget: "/users/1",
			deleteFunc: func(t *testing.T, ctx context.Context, id int64) error {
				t.Helper()

				if id != 1 {
					t.Fatalf("got %v, expected 1", id)
				}

				return internalErr
			},
			respStatusCode: http.StatusInternalServerError,
			wantErr: AppError{
				Error:   "internal_error",
				Message: "something went wrong",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()
			fakeService := &fakeService{
				deleteFunc: func(ctx context.Context, id int64) error {
					t.Helper()

					return tC.deleteFunc(t, ctx, id)
				},
			}
			resp := serveRequest(t, fakeService, http.MethodDelete, tC.reqTarget, nil)
			defer resp.Body.Close()

			assertStatus(t, resp, tC.respStatusCode)

			if resp.StatusCode == http.StatusNoContent {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("got unexpected read response body error: %v", err)
				}
				if len(body) != 0 {
					t.Fatalf("expected empty response body, but got %v", body)
				}
			} else {
				gotErr := decodeJSON[AppError](t, resp.Body)
				if gotErr != tC.wantErr {
					t.Fatalf("got AppError %v, expected %v", gotErr, tC.wantErr)
				}
			}
		})
	}
}

func serveRequest(
	t *testing.T,
	service UserService,
	method string,
	target string,
	body io.Reader,
) *http.Response {
	t.Helper()

	handler := NewUserHandler(service)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(method, target, body)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	return rec.Result()
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()

	if resp.StatusCode != want {
		t.Fatalf("got status %d, want %d", resp.StatusCode, want)
	}
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()

	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	return v
}
