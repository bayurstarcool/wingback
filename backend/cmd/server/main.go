package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/bayurstarcool/wingback/backend/internal/auth"
	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/db"
	"github.com/bayurstarcool/wingback/backend/internal/handlers"
	"github.com/bayurstarcool/wingback/backend/internal/hub"
	"github.com/bayurstarcool/wingback/backend/internal/location"
	"github.com/bayurstarcool/wingback/backend/internal/middleware"
	"github.com/bayurstarcool/wingback/backend/internal/repo"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	log.Printf("db: connected")

	r := repo.New(pool)
	h := hub.New()
	signer := auth.NewSigner(cfg.JWTSecret, 7*24*time.Hour)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(echomw.Recover())
	e.Use(echomw.Logger())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	authH := handlers.NewAuthHandler(r, signer)
	msgH := handlers.NewMessageHandlerWithResolver(cfg, r, h, location.NewNominatimResolver(cfg.GeocoderURL))

	api := e.Group("/api")
	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)
	api.GET("/carriers", msgH.ListCarriers)

	authed := api.Group("", middleware.JWT(signer))
	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/location", msgH.UpdateLocation)
	authed.GET("/users/search", msgH.SearchUsers)
	authed.POST("/messages", msgH.Compose)
	authed.GET("/messages/inbox", msgH.ListInbox)
	authed.GET("/messages/sent", msgH.ListSent)
	authed.GET("/messages/:id", msgH.GetMessage)
	authed.GET("/messages/:id/stream", msgH.Stream)

	// Serve the SvelteKit SPA build (adapter-static output). Mirrors
	// the Growly deployment: one Go binary serves both API and web.
	// When WEB_BUILD_DIR is empty the server is API-only (dev flow
	// where Vite serves the frontend on its own port).
	if cfg.WebBuildDir != "" {
		e.Static("/_app", cfg.WebBuildDir+"/_app")
		e.File("/robots.txt", cfg.WebBuildDir+"/robots.txt")
		e.GET("/*", func(c echo.Context) error {
			return c.File(cfg.WebBuildDir + "/index.html")
		})
		log.Printf("web: serving SPA from %s", cfg.WebBuildDir)
	}

	log.Printf("wingback backend listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
