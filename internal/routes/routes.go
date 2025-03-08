// Routes contains all of the routes for webdwelling
package routes

import (
    "fmt"
    "os"
    "strings"
    "strconv"
    "path/filepath"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

    "github.com/cjm-1/webdwelling/internal/auth"
    "github.com/cjm-1/webdwelling/internal/database"
)

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, templatesDir string, staticDir string) {
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

    // Bookmarks route
    e.GET("/bookmarks", auth.RequireAuth(func(c echo.Context) error {
        navItems := GetNavItems(templatesDir, true)
        userID := c.Get("user_id").(int)
        bookmarks, err := database.GetBookmarksByUserID(userID, true)

        if err != nil {
            return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
                "title": "Error",
                "NavItems": navItems,
                "ErrorCode": http.StatusInternalServerError,
                "ErrorMessage": err.Error(),
            })
        }

        folderCount := len(bookmarks.ChildFolders)
        bookmarkCount := len(bookmarks.ChildBookmarks)
        fmt.Println("Folder count: " + strconv.Itoa(folderCount) + ", bookmark count: " + strconv.Itoa(bookmarkCount))

        return c.Render(http.StatusOK, "bookmarks.html", map[string]interface{}{
            "title": "Bookmarks",
            "NavItems": navItems,
            "Bookmarks": bookmarks,
        })
    }))

    e.POST("/bookmarks/add-folder", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        name := c.FormValue("name")
        parentFolderID, err := strconv.Atoi(c.FormValue("parent_folder_id"))
        if err != nil {
            return err
        }
        // public := c.FormValue("public") == "on"
        var public bool = c.FormValue("public") == "true"

        // Ensure new folder belongs to current user
        if userID != c.Get("user_id").(int) {
            // Serve JSON error response
            return c.JSON(http.StatusUnauthorized, map[string]interface{}{
                "error": "You are not authorized to add a folder",
                "error_code": http.StatusUnauthorized,
            })
        }

        newFolder, err := database.AddBookmarkFolder(userID, name, &parentFolderID, public)
        if err != nil {
            // Serve JSON error response
            return c.JSON(http.StatusInternalServerError, map[string]interface{}{
                "error": err.Error(),
                "error_code": http.StatusInternalServerError,
            })
        }

        // Bookmark folder response type
        type BookmarkFolderResponse struct {
            ID int `json:"id"`
            Name string `json:"name"`
            ParentFolderID int `json:"parent_folder_id,omitempty"`
            CreatedAt string `json:"created_at"`
        }

        response := BookmarkFolderResponse{
            ID: newFolder.ID,
            Name: newFolder.Name,
            ParentFolderID: *newFolder.ParentFolderID,
            CreatedAt: newFolder.CreatedAt,
        }

        // Return the new folder details from newFolder but dereference pointers
        return c.JSON(http.StatusOK, response)
    }))

    e.Static("/", staticDir)

    e.POST("/login", auth.Login)

    e.GET("/logout", auth.Logout)

    // Blacklisted template names
    var blacklist []string = []string{"header", "footer", "error", "home", "bookmarks"}

    // Route all other templates
    for _, file := range files {
        var base string = filepath.Base(file)
        var name string = strings.TrimSuffix(base, ".html")
        var url string = "/" + name
        var blacklisted bool = false

        for _, b := range blacklist {
            if name == b {
                fmt.Println(name + " vs " + b)
                blacklisted = true
                break
            }
        }

        if blacklisted {
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

