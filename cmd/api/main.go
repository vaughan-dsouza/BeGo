package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vaughan-dsouza/BeGo/internal/db"
	"github.com/vaughan-dsouza/BeGo/internal/handlers"
	"github.com/vaughan-dsouza/BeGo/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := getenv("PORT", "4000")
	databaseURL := getenv("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	dbConn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer dbConn.Close()

	h := handlers.NewHandler(dbConn)

	r := chi.NewRouter()

	// Public
	r.Post("/signup", h.Auth.SignUp)
	r.Post("/login", h.Auth.Login)
	r.Post("/refresh", h.Auth.Refresh)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Get("/me", h.Auth.Me)
		r.Post("/logout", h.Auth.Logout)

		r.Get("/posts", h.Posts.GetPosts)
		r.Post("/posts", h.Posts.CreatePost)
		r.Get("/posts/{id}", h.Posts.GetPostByID)
		r.Put("/posts/{id}", h.Posts.UpdatePost)
		r.Delete("/posts/{id}", h.Posts.DeletePost)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
