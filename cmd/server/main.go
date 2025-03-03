package main

import (
    "path/filepath"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    "github.com/cjm-1/webdwelling/internal/database"
    "github.com/cjm-1/webdwelling/internal/routes"
)

func main() {
    e := echo.New()

    database.InitDB()

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())

    // Register routes
    routes.RegisterRoutes(e, filepath.Join("web", "templates"))

    e.Logger.Fatal(e.Start(":1323"))
}
