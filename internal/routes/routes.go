// Routes contains all of the routes for webdwelling
package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/labstack/echo/v4"
	"github.com/mead-one/webdwelling/internal/auth"
	"github.com/mead-one/webdwelling/internal/database"
)

const (
	maxBookmarkTitleLength = 256
	maxBookmarkURLLength   = 2048
	maxBookmarkTagsLength  = 256
	maxFolderNameLength    = 64
)

// Bookmark folder response type
type BookmarkFolderResponse struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ParentFolderID *int   `json:"parent_folder_id,omitempty"`
	Public         bool   `json:"public"`
	CreatedAt      string `json:"created_at"`
}

// Bookmark-related form values
type bookmarkFields struct {
	Title string
	URL   string
	Tags  string
}

// Folder-related form values
type folderFields struct {
	Name string
}

func RegisterRoutes(e *echo.Echo, basePath string, templatesDir string, staticDir string) {
	renderer := TemplateRenderer(templatesDir)
	e.Renderer = renderer

	files, _ := filepath.Glob(filepath.Join(templatesDir, "*.html"))

	var navItems []NavItem = buildNavItems(files)
	var navLoggedIn []NavItem = filterNavItems(navItems, true)
	var navLoggedOut []NavItem = filterNavItems(navItems, false)

	navFor := func(isAuthenticated bool) []NavItem {
		if isAuthenticated {
			return navLoggedIn
		}
		return navLoggedOut
	}

	// Set up language caser for casing titles
	caser := cases.Title(language.BritishEnglish)

	// Route / to rendered homepage
	e.GET("/", func(c echo.Context) error {
		var isAuthenticated bool = auth.IsUserAuthenticated(c)
		var username string
		var bookmarks database.BookmarkFolder
		var err error

		if isAuthenticated {
			username = c.Get("username").(string)
			bookmarks, err = database.GetBookmarksByUserID(c.Get("user_id").(int), true)
		}

		if err != nil {
			log.Printf("Failed to get bookmarks: %v", err)
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"BasePath":     basePath,
				"Title":        "Error",
				"NavItems":     navFor(isAuthenticated),
				"ErrorCode":    http.StatusInternalServerError,
				"ErrorMessage": "Failed to get bookmarks",
			})
		}

		return c.Render(http.StatusOK, "home.html", map[string]interface{}{
			"BasePath":        basePath,
			"Title":           "Home",
			"NavItems":        navFor(isAuthenticated),
			"IsAuthenticated": isAuthenticated,
			"Username":        username,
			"Bookmarks":       bookmarks,
		})
	})

	// Bookmarks route
	e.GET("/bookmarks", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		bookmarks, err := database.GetBookmarksByUserID(userID, true)
		if err != nil {
			log.Printf("Failed to get bookmarks: %v", err)
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"BasePath":     basePath,
				"Title":        "Error",
				"NavItems":     navFor(true),
				"ErrorCode":    http.StatusInternalServerError,
				"ErrorMessage": "Failed to get bookmarks",
			})
		}

		return c.Render(http.StatusOK, "bookmarks.html", map[string]interface{}{
			"BasePath":  basePath,
			"Title":     "Bookmarks",
			"NavItems":  navFor(true),
			"Bookmarks": bookmarks,
		})
	}))

	e.POST("/bookmarks/add-bookmark", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		var folderID *int

		fields, err := readBookmarkFields(c)
		if err != nil {
			log.Printf("Invalid bookmark fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":      err.Error(),
				"error_code": http.StatusBadRequest,
			})
		}

		if c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "" {
			folderID = nil
		} else {
			folderID = new(int)
			*folderID, err = strconv.Atoi(c.FormValue("folder_id"))
			if err != nil {
				log.Printf("Failed to parse folder ID: %v", err)
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":      "Invalid folder ID",
					"error_code": http.StatusBadRequest,
				})
			}
		}

		var public bool = c.FormValue("public") == "true"

		newBookmark, err := database.AddBookmark(userID, fields.Title, fields.URL, fields.Tags, folderID, public)
		if err != nil {
			log.Printf("Failed to add bookmark: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":      "Failed to add bookmark",
				"error_code": http.StatusInternalServerError,
			})
		}

		// Bookmark response type
		type BookmarkResponse struct {
			ID        int    `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Tags      string `json:"tags"`
			FolderID  *int   `json:"folder_id,omitempty"`
			Public    bool   `json:"public"`
			CreatedAt string `json:"created_at"`
		}

		response := BookmarkResponse{
			ID:        newBookmark.ID,
			Title:     newBookmark.Title,
			URL:       newBookmark.URL,
			Tags:      newBookmark.Tags,
			FolderID:  newBookmark.FolderID,
			Public:    newBookmark.Public,
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

	e.POST("/bookmarks/move-folder", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		folderID, err := strconv.Atoi(c.FormValue("folder_id"))
		if err != nil {
			return err
		}

		var parentFolderID *int

		if c.FormValue("parent_folder_id") == "null" || c.FormValue("parent_folder_id") == "" {
			parentFolderID = nil
		} else {
			parentFolderID = new(int)
			*parentFolderID, err = strconv.Atoi(c.FormValue("parent_folder_id"))
			if err != nil {
				return err
			}
		}

		err = database.MoveBookmarkFolder(userID, folderID, parentFolderID)
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

		fields, err := readBookmarkFields(c)
		if err != nil {
			log.Printf("Invalid bookmark fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":      err.Error(),
				"error_code": http.StatusBadRequest,
			})
		}

		var folderID *int
		if c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "" {
			folderID = nil
		} else {
			folderID = new(int)
			*folderID, err = strconv.Atoi(c.FormValue("folder_id"))
			if err != nil {
				return err
			}
		}

		var public bool = c.FormValue("public") == "true"

		newBookmark, err := database.EditBookmark(userID, bookmarkID, fields.Title, fields.URL, fields.Tags, folderID, public)
		if err != nil {
			log.Printf("Failed to edit bookmark: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":      "Failed to edit bookmark",
				"error_code": http.StatusInternalServerError,
			})
		}

		// Bookmark response type
		type BookmarkResponse struct {
			ID        int    `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Tags      string `json:"tags"`
			FolderID  *int   `json:"folder_id,omitempty"`
			Public    bool   `json:"public"`
			CreatedAt string `json:"created_at"`
		}

		response := BookmarkResponse{
			ID:        newBookmark.ID,
			Title:     newBookmark.Title,
			URL:       newBookmark.URL,
			Tags:      newBookmark.Tags,
			FolderID:  newBookmark.FolderID,
			Public:    newBookmark.Public,
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
		var parentFolderID *int

		fields, err := readFolderFields(c)
		if err != nil {
			log.Printf("Invalid folder fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":      err.Error(),
				"error_code": http.StatusBadRequest,
			})
		}

		if c.FormValue("parent_folder_id") == "null" || c.FormValue("parent_folder_id") == "" {
			parentFolderID = nil
		} else {
			parentFolderID = new(int)
			*parentFolderID, err = strconv.Atoi(c.FormValue("parent_folder_id"))
			if err != nil {
				return err
			}
		}

		var public bool = c.FormValue("public") == "true"

		newFolder, err := database.AddBookmarkFolder(userID, fields.Name, parentFolderID, public)
		if err != nil {
			log.Printf("Failed to add folder: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":      "Failed to add folder",
				"error_code": http.StatusInternalServerError,
			})
		}

		response := BookmarkFolderResponse{
			ID:             newFolder.ID,
			Name:           newFolder.Name,
			ParentFolderID: newFolder.ParentFolderID,
			Public:         newFolder.Public,
			CreatedAt:      newFolder.CreatedAt,
		}

		// Return the new folder details from newFolder but dereference pointers
		return c.JSON(http.StatusOK, response)
	}))

	e.POST("/bookmarks/rename-folder", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		folderID, err := strconv.Atoi(c.FormValue("folder_id"))
		if err != nil {
			return err
		}

		fields, err := readFolderFields(c)
		if err != nil {
			log.Printf("Invalid folder fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":      err.Error(),
				"error_code": http.StatusBadRequest,
			})
		}

		newFolder, err := database.RenameBookmarkFolder(userID, folderID, fields.Name)
		if err != nil {
			return err
		}

		response := BookmarkFolderResponse{
			ID:             newFolder.ID,
			Name:           newFolder.Name,
			ParentFolderID: newFolder.ParentFolderID,
			Public:         newFolder.Public,
			CreatedAt:      newFolder.CreatedAt,
		}
		return c.JSON(http.StatusOK, response)
	}))

	e.POST("/bookmarks/delete-folder", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		folderID, err := strconv.Atoi(c.FormValue("folder_id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid folder ID",
			})
		}

		err = database.DeleteBookmarkFolder(userID, folderID)
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Folder not found",
			})
		} else if err != nil {
			log.Printf("Failed to delete folder %d for user %d: %v", folderID, userID, err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Failed to delete folder",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}))

	e.Static("/", staticDir)

	e.POST("/login", auth.Login, middleware.RateLimiter(
		middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate: 1.0 / 12, Burst: 5, ExpiresIn: 10 * time.Minute,
			},
		),
	))

	e.POST("/logout", auth.Logout)

	// Blacklisted template names
	var blacklist []string = []string{"header", "footer", "error", "home", "bookmarks"}

	// Route all other templates
	for _, file := range files {
		var base string = filepath.Base(file)
		var name string = strings.TrimSuffix(base, ".html")
		var title string = caser.String(name)
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

		// Read and parse metadata comment
		line, err := getNavComment(f)
		f.Close()
		if err != nil {
			fmt.Println(err)
			continue
		}
		_, _, requireAuth, hideIfAuth := parseNavComment(line)

		if requireAuth {
			e.GET(url, auth.RequireAuth(func(c echo.Context) error {
				return c.Render(http.StatusOK, name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    title,
					"NavItems": navFor(true),
				})
			}))
		} else if hideIfAuth {
			e.GET(url, auth.RequireNoAuth(func(c echo.Context) error {
				return c.Render(http.StatusOK, name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    title,
					"NavItems": navFor(false),
				})
			}))
		} else {
			e.GET(url, func(c echo.Context) error {
				var isAuthenticated bool = auth.IsUserAuthenticated(c)
				return c.Render(http.StatusOK, name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    title,
					"NavItems": navFor(isAuthenticated),
				})
			})
		}
	}
}

func normaliseBookmarkURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	addedScheme := !strings.Contains(raw, "://")
	if addedScheme {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Host == "" {
		return "", errors.New("invalid bookmark URL")
	}

	if addedScheme && parsed.Hostname() == "localhost" {
		parsed.Scheme = "http"
	}

	return parsed.String(), nil
}

// Trim form values, validate the URL and ensure acceptable lengths for new values
func readBookmarkFields(c echo.Context) (bookmarkFields, error) {
	fields := bookmarkFields{
		Title: strings.TrimSpace(c.FormValue("title")),
		Tags:  strings.TrimSpace(c.FormValue("tags")),
	}

	if length := utf8.RuneCountInString(fields.Title); length == 0 || length > maxBookmarkTitleLength {
		return fields, fmt.Errorf("title must be 1 to %d characters", maxBookmarkTitleLength)
	}
	if utf8.RuneCountInString(fields.Tags) > maxBookmarkTagsLength {
		return fields, fmt.Errorf("tags must be at most %d characters", maxBookmarkTagsLength)
	}

	bookmarkURL, err := normaliseBookmarkURL(c.FormValue("url"))
	if err != nil {
		return fields, errors.New("URL must be valid, and start with http:// or https://, or omit the scheme entirely")
	}
	if utf8.RuneCountInString(bookmarkURL) > maxBookmarkURLLength {
		return fields, fmt.Errorf("URL must be 1 to %d characters", maxBookmarkURLLength)
	}
	fields.URL = bookmarkURL

	return fields, nil
}

func readFolderFields(c echo.Context) (folderFields, error) {
	fields := folderFields{
		Name: strings.TrimSpace(c.FormValue("name")),
	}

	if length := utf8.RuneCountInString(fields.Name); length == 0 || length > maxFolderNameLength {
		return fields, fmt.Errorf("name must be 1 to %d characters", maxFolderNameLength)
	}

	return fields, nil
}
