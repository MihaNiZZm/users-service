package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

type AppError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type UserHandler struct {
	service *Service
}

func NewUserHandler(service *Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", h.Create)
	mux.HandleFunc("GET /users", h.List)
	mux.HandleFunc("GET /users/{id}", h.Get)
	mux.HandleFunc("DELETE /users/{id}", h.Delete)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "id must be a number")
		return
	}

	user, err := h.service.Get(r.Context(), id)
	if err != nil {
		mapAndRespondError(w, err)
		return
	}

	respondWithJSONEntity(w, http.StatusOK, user)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var userRequest UserRequest
	err := json.NewDecoder(r.Body).Decode(&userRequest)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request_body", "request body must be a json with format { name: 'string', age: 20 }")
		return
	}

	user, err := h.service.Create(r.Context(), userRequest)
	if err != nil {
		mapAndRespondError(w, err)
		return
	}

	respondWithJSONEntity(w, http.StatusCreated, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetAll(r.Context())
	if err != nil {
		mapAndRespondError(w, err)
		return
	}

	respondWithJSONEntity(w, http.StatusOK, users)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "id must be a number")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		mapAndRespondError(w, err)
		return
	}

	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func respondWithJSONEntity(w http.ResponseWriter, code int, entity any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(entity)
}

func respondWithError(w http.ResponseWriter, code int, errType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	appErr := AppError{
		Error:   errType,
		Message: message,
	}

	_ = json.NewEncoder(w).Encode(appErr)
}

func mapAndRespondError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrEmptyUserName) {
		respondWithError(w, http.StatusBadRequest, "invalid_name", err.Error())
		return
	}
	if errors.Is(err, ErrInvalidUserAge) {
		respondWithError(w, http.StatusBadRequest, "invalid_age", err.Error())
		return
	}

	var userNotFound *ErrUserNotFound
	if errors.As(err, &userNotFound) {
		respondWithError(w, http.StatusNotFound, "not_found", userNotFound.Error())
		return
	}

	respondWithError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
}
