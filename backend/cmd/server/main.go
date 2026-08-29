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
	msgH := handlers.NewMessageHandler(cfg, r, h)

	api := e.Group("/api")
	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)
	api.GET("/carriers", msgH.ListCarriers)

	authed := api.Group("", middleware.JWT(signer))
	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/location", msgH.UpdateLocation)
	authed.POST("/messages", msgH.Compose)
	authed.GET("/messages/inbox", msgH.ListInbox)
	authed.GET("/messages/sent", msgH.ListSent)
	authed.GET("/messages/:id", msgH.GetMessage)
	authed.GET("/messages/:id/stream", msgH.Stream)

	log.Printf("wingback backend listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
