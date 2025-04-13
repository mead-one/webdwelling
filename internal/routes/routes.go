// Routes contains all of the routes for webdwelling
package routes

import (
    "fmt"
    "os"
    "strings"
    "strconv"
    "path/filepath"

    "golang.org/x/exp/slices"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

    "github.com/cjm-1/webdwelling/internal/auth"
    "github.com/cjm-1/webdwelling/internal/database"
)

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

// Bookmark folder response type
type BookmarkFolderResponse struct {
    ID int `json:"id"`
    Name string `json:"name"`
    ParentFolderID *int `json:"parent_folder_id,omitempty"`
    Public bool `json:"public"`
    CreatedAt string `json:"created_at"`
}

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

        return c.Render(http.StatusOK, "bookmarks.html", map[string]interface{}{
            "title": "Bookmarks",
            "NavItems": navItems,
            "Bookmarks": bookmarks,
        })
    }))

    e.POST("/bookmarks/add-bookmark", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        title := c.FormValue("title")
        url := c.FormValue("url")
        tags := c.FormValue("tags")
        var folderID *int
        var err error
        if (c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "") {
            folderID = nil
        } else {
            folderID = new(int)
            *folderID, err = strconv.Atoi(c.FormValue("folder_id"))
            if err != nil {
                return c.JSON(http.StatusBadRequest, map[string]interface{}{
                    "error": "Invalid folder ID",
                    "error_code": http.StatusBadRequest,
                })
            }
        }

        var public bool = c.FormValue("public") == "true"

        newBookmark, err := database.AddBookmark(userID, title, url, tags, folderID, public)
        if err != nil {
            // Serve JSON error response
            return c.JSON(http.StatusInternalServerError, map[string]interface{}{
                "error": err.Error(),
                "error_code": http.StatusInternalServerError,
            })
        }

        // Bookmark response type
        type BookmarkResponse struct {
            ID int `json:"id"`
            Title string `json:"title"`
            URL string `json:"url"`
            Tags string `json:"tags"`
            FolderID *int `json:"folder_id,omitempty"`
            Public bool `json:"public"`
            CreatedAt string `json:"created_at"`
        }

        response := BookmarkResponse{
            ID: newBookmark.ID,
            Title: newBookmark.Title,
            URL: newBookmark.URL,
            Tags: newBookmark.Tags,
            FolderID: newBookmark.FolderID,
            Public: newBookmark.Public,
            CreatedAt: newBookmark.CreatedAt,
        }

        // Return the new bookmark details from newBookmark but dereference pointers
        return c.JSON(http.StatusOK, response)
    }))

    e.POST("/bookmarks/move-bookmark", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        bookmarkID, err := strconv.Atoi(c.FormValue("bookmark_id"))
        if err != nil {
            return err
        }

		var folderIDValue string = c.FormValue("folder_id")

		var folderID *int
		if folderIDValue == "" {
			folderID = nil
		} else {
			folderIDConv, err := strconv.Atoi(folderIDValue)
			if err != nil {
				return err
			}
			folderID = &folderIDConv
		}

        err = database.MoveBookmark(userID, bookmarkID, folderID)
        if err != nil {
            return err
        }
        
        return c.JSON(http.StatusOK, map[string]interface{}{
            "success": true,
        })
    }))

    e.POST("/bookmarks/edit-bookmark", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        bookmarkID, err := strconv.Atoi(c.FormValue("bookmark_id"))
        if err != nil {
            return err
        }

        title := c.FormValue("title")
        url := c.FormValue("url")
        tags := c.FormValue("tags")
        var folderID *int
        if (c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "") {
            folderID = nil
        } else {
            folderID = new(int)
            *folderID, err = strconv.Atoi(c.FormValue("folder_id"))
            if err != nil {
                return err
            }
        }

        var public bool = c.FormValue("public") == "true"
        fmt.Println("Form values:", c.FormValue("title"), c.FormValue("url"), c.FormValue("tags"), c.FormValue("folder_id"), c.FormValue("public"))
        fmt.Println("Parameter values:", title, url, tags, folderID, public)

        newBookmark, err := database.EditBookmark(userID, bookmarkID, title, url, tags, folderID, public)
        if err != nil {
            // Serve JSON error response
            return c.JSON(http.StatusInternalServerError, map[string]interface{}{
                "error": err.Error(),
                "error_code": http.StatusInternalServerError,
            })
        }

        // Bookmark response type
        type BookmarkResponse struct {
            ID int `json:"id"`
            Title string `json:"title"`
            URL string `json:"url"`
            Tags string `json:"tags"`
            FolderID *int `json:"folder_id,omitempty"`
            Public bool `json:"public"`
            CreatedAt string `json:"created_at"`
        }

        response := BookmarkResponse{
            ID: newBookmark.ID,
            Title: newBookmark.Title,
            URL: newBookmark.URL,
            Tags: newBookmark.Tags,
            FolderID: newBookmark.FolderID,
            Public: newBookmark.Public,
            CreatedAt: newBookmark.CreatedAt,
        }

        // Return the new bookmark details from newBookmark but dereference pointers
        return c.JSON(http.StatusOK, response)
    }))

    e.POST("/bookmarks/delete-bookmark", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        bookmarkID, err := strconv.Atoi(c.FormValue("bookmark_id"))
        if err != nil {
            return err
        }

        err = database.DeleteBookmark(userID, bookmarkID)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusOK, map[string]interface{}{
            "success": true,
        })
    }))

    e.POST("/bookmarks/add-folder", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        name := c.FormValue("name")
        var parentFolderID *int
        var err error
        if (c.FormValue("parent_folder_id") == "null" || c.FormValue("parent_folder_id") == "") {
            parentFolderID = nil
        } else {
            parentFolderID = new(int)
            *parentFolderID, err = strconv.Atoi(c.FormValue("parent_folder_id"))
            if err != nil {
                return err
            }
        }

        var public bool = c.FormValue("public") == "true"

        newFolder, err := database.AddBookmarkFolder(userID, name, parentFolderID, public)
        if err != nil {
            // Serve JSON error response
            return c.JSON(http.StatusInternalServerError, map[string]interface{}{
                "error": err.Error(),
                "error_code": http.StatusInternalServerError,
            })
        }

        response := BookmarkFolderResponse{
            ID: newFolder.ID,
            Name: newFolder.Name,
            ParentFolderID: newFolder.ParentFolderID,
            Public: newFolder.Public,
            CreatedAt: newFolder.CreatedAt,
        }

        // Return the new folder details from newFolder but dereference pointers
        return c.JSON(http.StatusOK, response)
    }))

    e.POST("/bookmarks/move-folder", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        folderID, err := strconv.Atoi(c.FormValue("folder_id"))
        if err != nil {
            return err
        }

        parentFolderID, err := strconv.Atoi(c.FormValue("parent_folder_id"))
        if err != nil {
            return err
        }

        err = database.MoveBookmarkFolder(userID, folderID, &parentFolderID)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusOK, map[string]interface{}{
            "success": true,
        })
    }))

    e.POST("/bookmarks/rename-folder", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        folderID, err := strconv.Atoi(c.FormValue("folder_id"))
        if err != nil {
            return err
        }

        name := c.FormValue("name")

        newFolder, err := database.RenameBookmarkFolder(userID, folderID, name)
        if err != nil {
            return err
        }

        response := BookmarkFolderResponse{
            ID: newFolder.ID,
            Name: newFolder.Name,
            ParentFolderID: newFolder.ParentFolderID,
            Public: newFolder.Public,
            CreatedAt: newFolder.CreatedAt,
        }
        return c.JSON(http.StatusOK, response)
    }))

    e.POST("/bookmarks/delete-folder", auth.RequireAuth(func(c echo.Context) error {
        userID := c.Get("user_id").(int)
        folderID, err := strconv.Atoi(c.FormValue("folder_id"))
        if err != nil {
            return err
        }

        err = database.DeleteBookmarkFolder(userID, folderID)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusOK, map[string]interface{}{
            "success": true,
        })
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

        if slices.Contains(blacklist, name) {
            blacklisted = true
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

