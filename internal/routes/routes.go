// Routes contains all of the routes for webdwelling
package routes

import (
    "fmt"
    "os"
    "strings"
    "path/filepath"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

    "github.com/cjm-1/webdwelling/internal/auth"
)

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, templatesDir string) {
    renderer := TemplateRenderer(templatesDir)
    e.Renderer = renderer
    files, _ := filepath.Glob(filepath.Join(templatesDir, "*.html"))

    // Set up language caser for casing titles
    caser := cases.Title(language.BritishEnglish)

    // Route / to rendered homepage
    e.GET("/", func(c echo.Context) error {
        var isAuthenticated bool = auth.IsUserAuthenticated(c)
        navItems := GetNavItems(templatesDir, isAuthenticated)
        username := c.Get("username")

        return c.Render(http.StatusOK, "home.html", map[string]interface{}{
            "title": "Home",
            "NavItems": navItems,
            "IsAuthenticated": isAuthenticated,
            "Username": username,
        })
    })

    e.POST("/login", auth.Login)

    e.GET("/logout", auth.Logout)

    // Route all other templates
    for _, file := range files {
        var base string = filepath.Base(file)
        var name string = strings.TrimSuffix(base, ".html")
        var url string = "/" + name

        if name == "home" || name == "header" || name == "footer" {
            continue
        }

        // Open file
        f, err := os.Open(file)
        if err != nil {
            continue
        }
        defer f.Close()

        // Read and parse metadata comment
        line, err := getNavComment(f)
        if err != nil {
            fmt.Println(err)
            continue
        }
        _, _, requireAuth, hideIfAuth := parseNavComment(line)

        if requireAuth {
            e.GET(url, auth.RequireAuth(func(c echo.Context) error {
                var isAuthenticated bool = auth.IsUserAuthenticated(c)
                navItems := GetNavItems(templatesDir, isAuthenticated)
                return c.Render(http.StatusOK, name + ".html", map[string]interface{}{
                    "title": caser.String(name),
                    "NavItems": navItems,
                })
            }))
        } else if hideIfAuth {
            e.GET(url, auth.RequireNoAuth(func(c echo.Context) error {
                var isAuthenticated bool = auth.IsUserAuthenticated(c)
                navItems := GetNavItems(templatesDir, isAuthenticated)
                return c.Render(http.StatusOK, name + ".html", map[string]interface{}{
                    "title": caser.String(name),
                    "NavItems": navItems,
                })
            }))
        } else {
            e.GET(url, func(c echo.Context) error {
                var isAuthenticated bool = auth.IsUserAuthenticated(c)
                navItems := GetNavItems(templatesDir, isAuthenticated)
                return c.Render(http.StatusOK, name + ".html", map[string]interface{}{
                    "title": caser.String(name),
                    "NavItems": navItems,
                })
            })
        }
    }
}

