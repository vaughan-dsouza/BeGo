package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/vaughan-dsouza/BeGo/internal/models"
	"github.com/vaughan-dsouza/BeGo/internal/utils"
)

type PostHandler struct {
	DB *sqlx.DB
}

func NewPostHandler(db *sqlx.DB) *PostHandler {
	return &PostHandler{DB: db}
}

// ---------------------- CREATE ----------------------

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := utils.DecodeJSON(w, r, &body); err != nil {
		return
	}

	userID := r.Context().Value(utils.CtxUserIDKey).(int64)

	query := `
        INSERT INTO posts (user_id, title, content)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at
    `

	post := models.Post{
		UserID:  userID,
		Title:   body.Title,
		Content: body.Content,
	}

	err := h.DB.QueryRowx(query, userID, body.Title, body.Content).
		Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, post)
}

// ---------------------- GET ONE ----------------------

func (h *PostHandler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var post models.Post

	err := h.DB.Get(&post, `SELECT * FROM posts WHERE id=$1`, id)
	if err != nil {
		utils.JSONError(w, http.StatusNotFound, "Post not found")
		return
	}

	utils.JSON(w, http.StatusOK, post)
}

// ---------------------- LIST ----------------------

func (h *PostHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	var posts []models.Post

	err := h.DB.Select(&posts, `SELECT * FROM posts ORDER BY created_at DESC`)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, posts)
}

// ---------------------- UPDATE ----------------------

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	var body struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}

	if err := utils.DecodeJSON(w, r, &body); err != nil {
		return
	}

	var post models.Post
	err := h.DB.Get(&post, `SELECT * FROM posts WHERE id=$1`, id)
	if err != nil {
		utils.JSONError(w, http.StatusNotFound, "Post not found")
		return
	}

	if body.Title != nil {
		post.Title = *body.Title
	}
	if body.Content != nil {
		post.Content = *body.Content
	}
	post.UpdatedAt = time.Now()

	_, err = h.DB.Exec(`
        UPDATE posts 
        SET title=$1, content=$2, updated_at=$3
        WHERE id=$4
    `, post.Title, post.Content, post.UpdatedAt, id)

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, post)
}

// ---------------------- DELETE ----------------------

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_, err := h.DB.Exec(`DELETE FROM posts WHERE id=$1`, id)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
