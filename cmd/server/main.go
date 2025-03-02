package main

import (
    "github.com/labstack/echo/v4"

    "github.com/cjm-1/webdwelling/internal/database"
    "github.com/cjm-1/webdwelling/internal/auth"
    "github.com/cjm-1/webdwelling/internal/routes"
)

func main() {
    e := echo.New()

    database.InitDB()

    e.GET("/", routes.Index)
    // e.GET("/login", routes.Login)
    e.POST("/login", auth.Login)
    e.GET("/logout", auth.Logout)
    e.GET("/bookmarks", routes.Bookmarks)
    // e.POST("/bookmarks/create", auth.RequireAuth, routes.CreateBookmark)


    e.Logger.Fatal(e.Start(":1323"))
}
