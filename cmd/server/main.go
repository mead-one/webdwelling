package main

import (
    "path/filepath"
    "log"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

	"github.com/mead-one/webdwelling/config"
    "github.com/mead-one/webdwelling/internal/database"
    "github.com/mead-one/webdwelling/internal/routes"
)

func main() {
	cfg, err := config.LoadConfig("/etc/webdwelling/config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

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

    e.Logger.Fatal(e.Start(":1323"))
}
