package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bayurstarcool/wingback/backend/internal/auth"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
)

type AuthHandler struct {
	Repo   *repo.Repo
	Signer *auth.Signer
}

func NewAuthHandler(r *repo.Repo, s *auth.Signer) *AuthHandler {
	return &AuthHandler{Repo: r, Signer: s}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type authResponse struct {
	Token       string `json:"token"`
	ExpiresAt   string `json:"expires_at"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email, password, and display_name are required")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "hash password")
	}

	u, err := h.Repo.CreateUser(c.Request().Context(), req.Email, string(hash), req.DisplayName)
	if err != nil {
		// Most likely a unique violation on email — surface as 409.
		if strings.Contains(err.Error(), "users_email_key") {
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	tok, exp, err := h.Signer.Issue(u.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "issue token")
	}

	return c.JSON(http.StatusCreated, authResponse{
		Token:       tok,
		ExpiresAt:   exp.UTC().Format("2006-01-02T15:04:05Z"),
		UserID:      u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	tok, exp, err := h.Signer.Issue(u.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "issue token")
	}

	return c.JSON(http.StatusOK, authResponse{
		Token:       tok,
		ExpiresAt:   exp.UTC().Format("2006-01-02T15:04:05Z"),
		UserID:      u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
	})
}

func (h *AuthHandler) Me(c echo.Context) error {
	uid, _ := c.Get("user_id").(string)
	u, err := h.Repo.GetUserByID(c.Request().Context(), uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, map[string]any{
		"user_id":      u.ID,
		"email":        u.Email,
		"display_name": u.DisplayName,
		"currency":     u.Currency,
		"last_lat":     u.LastLat,
		"last_lng":     u.LastLng,
	})
}
