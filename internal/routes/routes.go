// Routes contains all of the routes for webdwelling
package routes

import (
    "github.com/cjm-1/webdwelling/internal/auth"
)

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, templatesDir string) {
    renderer := TemplateRenderer(templatesDir)
    e.Renderer = renderer

    navItems := GetNavItems()

    // Route / to rendered homepage
    e.GET("/", func(c echo.Context) error {
        return c.Render(http.StatusOK, "home.html", map[string]interface{}{
            "title": "Home",
            "NavItems": navItems,
        })
    })

    e.GET("/login", func(c echo.Context) error {
        return c.Render(http.StatusOK, "login.html", map[string]interface{}{
            "title": "Login",
            "NavItems": navItems,
        })
    })

    e.POST("/login", auth.Login)

    // Route all nav items to rendered nav item
    // for _, navItem := range navItems {
    //     e.GET(navItem.URL, func(c echo.Context) error {
    //         return c.Render(http.StatusOK, navItem.Name + ".html", map[string]interface{}{
    //             "title": navItem.Name,
    //             "NavItems": navItems,
    //         })
    //     })
    // }
}

