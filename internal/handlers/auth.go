package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/vaughan-dsouza/BeGo/internal/models"
	"github.com/vaughan-dsouza/BeGo/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *sqlx.DB
}

// FIX: return *AuthHandler
func NewAuthHandler(db *sqlx.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

// ----------- Request/Response DTOs -------------

type signUpReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ----------- Helper: Write JSON -------------

func (h *AuthHandler) json(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) error(w http.ResponseWriter, code int, msg string) {
	h.json(w, code, map[string]string{"error": msg})
}

// -------------- SIGN UP ----------------------

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Email == "" || req.Password == "" {
		h.error(w, http.StatusBadRequest, "email and password required")
		return
	}

	if len(req.Password) < 6 {
		h.error(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "internal error")
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
	`, req.Email, string(hash))

	if err != nil {
		h.error(w, http.StatusBadRequest, "email already exists")
		return
	}

	h.json(w, http.StatusCreated, map[string]string{
		"message": "user created",
	})
}

// -------------- LOGIN ------------------------

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid json")
		return
	}

	var u models.User

	err := h.DB.Get(&u, `
		SELECT id, email, password_hash, role
		FROM users
		WHERE email=$1
	`, req.Email)

	if errors.Is(err, sql.ErrNoRows) {
		h.error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		h.error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	access, expAccess, err := utils.GenerateToken(u.ID, u.Email, os.Getenv("ACCESS_SECRET"), os.Getenv("ACCESS_TTL"))
	if err != nil {
		h.error(w, http.StatusInternalServerError, "token error")
		return
	}

	refresh, expRefresh, err := utils.GenerateToken(u.ID, u.Email, os.Getenv("REFRESH_SECRET"), os.Getenv("REFRESH_TTL"))
	if err != nil {
		h.error(w, http.StatusInternalServerError, "token error")
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, u.ID, refresh, time.Unix(expRefresh, 0))

	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	h.json(w, http.StatusOK, tokenResp{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expAccess,
	})
}

// ---------------- REFRESH ---------------------

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid json")
		return
	}

	claims, err := utils.VerifyToken(req.RefreshToken, os.Getenv("REFRESH_SECRET"))
	if err != nil {
		h.error(w, http.StatusUnauthorized, "invalid token")
		return
	}

	var exists bool
	err = h.DB.Get(&exists, `
		SELECT EXISTS (
			SELECT 1 FROM refresh_tokens
			WHERE token=$1 AND user_id=$2 AND expires_at > NOW()
		)
	`, req.RefreshToken, claims.SubjectInt())

	if err != nil || !exists {
		h.error(w, http.StatusUnauthorized, "refresh token expired or invalid")
		return
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback()

	_, _ = tx.Exec(`DELETE FROM refresh_tokens WHERE token=$1`, req.RefreshToken)

	access, expAccess, err := utils.GenerateToken(claims.SubjectInt(), claims.Email, os.Getenv("ACCESS_SECRET"), os.Getenv("ACCESS_TTL"))
	if err != nil {
		h.error(w, http.StatusInternalServerError, "token error")
		return
	}

	refresh, expRefresh, err := utils.GenerateToken(claims.SubjectInt(), claims.Email, os.Getenv("REFRESH_SECRET"), os.Getenv("REFRESH_TTL"))
	if err != nil {
		h.error(w, http.StatusInternalServerError, "token error")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, claims.SubjectInt(), refresh, time.Unix(expRefresh, 0))

	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := tx.Commit(); err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	h.json(w, http.StatusOK, tokenResp{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    expAccess,
	})
}

// -------------- LOGOUT -----------------------

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid json")
		return
	}

	_, err := h.DB.Exec(`DELETE FROM refresh_tokens WHERE token=$1`, req.RefreshToken)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -------------- ME (protected) ----------------

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value(utils.CtxUserIDKey).(int64)
	if !ok {
		h.error(w, http.StatusUnauthorized, "not authorized")
		return
	}

	var user models.User
	err := h.DB.Get(&user, `
		SELECT id, email, role, created_at
		FROM users
		WHERE id=$1
	`, uid)

	if err != nil {
		h.error(w, http.StatusInternalServerError, "db error")
		return
	}

	h.json(w, http.StatusOK, user)
}
