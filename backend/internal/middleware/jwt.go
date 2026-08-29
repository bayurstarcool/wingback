package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/bayurstarcool/wingback/backend/internal/auth"
)

const userIDKey = "user_id"

// JWT returns Echo middleware that requires a valid token and stashes
// the user ID on the context. The token can come from either an
// `Authorization: Bearer <jwt>` header OR a `?token=<jwt>` query string
// (the latter is needed because browsers can't attach custom headers
// to a WebSocket handshake).
func JWT(signer *auth.Signer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := extractToken(c)
			if raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}
			claims, err := signer.Parse(raw)
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set(userIDKey, claims.UserID)
			return next(c)
		}
	}
}

func extractToken(c echo.Context) string {
	if h := c.Request().Header.Get(echo.HeaderAuthorization); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	return c.QueryParam("token")
}

// UserID returns the authenticated user ID stored on the context, or
// the empty string if no JWT was required for the route.
func UserID(c echo.Context) string {
	v, _ := c.Get(userIDKey).(string)
	return v
}
