package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MihaNiZZm/users-service/internal/httpapi"
	"github.com/MihaNiZZm/users-service/internal/user"
)

func main() {
	repository := user.NewUserRepository()
	service := user.NewUserService(repository)
	handler := httpapi.NewUserHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := http.Server{
		Addr:              ":8080",
		Handler:           mux,
		IdleTimeout:       10 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("server stopped:", err.Error())
		}
	})
	fmt.Println("server is listening on http://localhost:8080")

	wg.Wait()
	fmt.Println("server is shut down")
}
