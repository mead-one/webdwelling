package main

import (
    "flag"
    "fmt"
    "path/filepath"
    "log"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

	"github.com/mead-one/webdwelling/config"
    "github.com/mead-one/webdwelling/internal/auth"
    "github.com/mead-one/webdwelling/internal/database"
    "github.com/mead-one/webdwelling/internal/routes"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "path to the config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	auth.JWTSecret = cfg.JWTSecret

    e := echo.New()

    e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
    	return func(c echo.Context) error {
    		c.Set("basePath", cfg.BasePath)
    		return next(c)
    	}
    })

    database.InitDB()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    // Register routes
    routes.RegisterRoutes(e, cfg.BasePath, filepath.Join("web", "templates"), filepath.Join("web", "static"))

    e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Port)))
}
