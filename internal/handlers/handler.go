package handlers

import "github.com/jmoiron/sqlx"

type Handler struct {
	DB    *sqlx.DB
	Auth  *AuthHandler
	Posts *PostHandler
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{
		DB:    db,
		Auth:  NewAuthHandler(db),
		Posts: NewPostHandler(db),
	}
}
