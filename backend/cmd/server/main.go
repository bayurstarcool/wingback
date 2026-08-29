package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/bayurstarcool/wingback/backend/internal/config"
	"github.com/bayurstarcool/wingback/backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	msgHandler := handlers.NewMessageHandler(cfg)
	api := e.Group("/api")
	api.POST("/messages", msgHandler.Compose)

	log.Printf("wingback backend listening on :%s (env=%s)", cfg.Port, cfg.Env)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
