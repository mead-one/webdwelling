// Routes contains all of the routes for webdwelling
package routes

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	ParentFolderID *int64 `json:"parent_folder_id,omitempty"`
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

func RegisterRoutes(e *echo.Echo, basePath string, templatesDir string, staticDir string) error {
	pages, err := loadPages(templatesDir)
	if err != nil {
		return err
	}
	loginPage, ok := pages["login"]
	if !ok {
		return errors.New(`template "login.html" not found`)
	}
	renderer := TemplateRenderer(templatesDir)
	e.Renderer = renderer

	var navItems []NavItem = buildNavItems(pages)
	var navLoggedIn []NavItem = filterNavItems(navItems, true)
	var navLoggedOut []NavItem = filterNavItems(navItems, false)

	navFor := func(isAuthenticated bool) []NavItem {
		if isAuthenticated {
			return navLoggedIn
		}
		return navLoggedOut
	}

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
		var folderID *int64

		fields, err := readBookmarkFields(c)
		if err != nil {
			log.Printf("Invalid bookmark fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}

		if c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "" {
			folderID = nil
		} else {
			folderID = new(int64)
			*folderID, err = strconv.ParseInt(c.FormValue("folder_id"), 10, 64)
			if err != nil {
				log.Printf("Failed to parse folder ID: %v", err)
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error":   "Invalid folder ID",
				})
			}
		}

		var public bool = c.FormValue("public") == "true"

		newBookmark, err := database.AddBookmark(userID, fields.Title, fields.URL, fields.Tags, folderID, public)
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to add bookmark: %v", err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Destination folder not found",
			})
		}
		if err != nil {
			log.Printf("Failed to add bookmark: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Failed to add bookmark",
			})
		}

		// Bookmark response type
		type BookmarkResponse struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Tags      string `json:"tags"`
			FolderID  *int64 `json:"folder_id,omitempty"`
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
		bookmarkID, err := strconv.ParseInt(c.FormValue("bookmark_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid bookmark ID",
			})
		}

		var folderIDValue string = c.FormValue("folder_id")

		var folderID *int64
		if folderIDValue == "" {
			folderID = nil
		} else {
			folderIDConv, err := strconv.ParseInt(folderIDValue, 10, 64)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error":   "Invalid folder ID",
				})
			}
			folderID = &folderIDConv
		}

		err = database.MoveBookmark(userID, bookmarkID, folderID)
		if errors.Is(err, database.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Bookmark or folder not found",
			})
		}
		if err != nil {
			log.Printf("Failed to move bookmark: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Something went wrong",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}))

	e.POST("/bookmarks/move-folder", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		folderID, err := strconv.ParseInt(c.FormValue("folder_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid folder ID",
			})
		}

		var parentFolderID *int64

		if c.FormValue("parent_folder_id") == "null" || c.FormValue("parent_folder_id") == "" {
			parentFolderID = nil
		} else {
			parentFolderID = new(int64)
			*parentFolderID, err = strconv.ParseInt(c.FormValue("parent_folder_id"), 10, 64)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error":   "Invalid parent folder ID",
				})
			}
		}

		err = database.MoveBookmarkFolder(userID, folderID, parentFolderID)
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to move folder %d: %v", folderID, err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Destination folder not found",
			})
		} else if errors.Is(err, database.ErrInvalidMove) {
			log.Printf("Failed to move folder %d: %v", folderID, err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "A folder can't be moved into one of its own subfolders",
			})
		} else if errors.Is(err, database.ErrDuplicateName) {
			log.Printf("Failed to move folder %d: %v", folderID, err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "A folder of that name already exists there",
			})
		} else if err != nil {
			log.Printf("Failed to move folder %d: %v", folderID, err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Something went wrong",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}))

	e.POST("/bookmarks/edit-bookmark", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		bookmarkID, err := strconv.ParseInt(c.FormValue("bookmark_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid bookmark ID",
			})
		}

		fields, err := readBookmarkFields(c)
		if err != nil {
			log.Printf("Invalid bookmark fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}

		var folderID *int64
		if c.FormValue("folder_id") == "null" || c.FormValue("folder_id") == "" {
			folderID = nil
		} else {
			folderID = new(int64)
			*folderID, err = strconv.ParseInt(c.FormValue("folder_id"), 10, 64)
			if err != nil {
				log.Printf("Invalid folder ID: %v", err)
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error":   "Invalid folder ID",
				})
			}
		}

		var public bool = c.FormValue("public") == "true"

		newBookmark, err := database.EditBookmark(userID, bookmarkID, fields.Title, fields.URL, fields.Tags, folderID, public)
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to edit bookmark: %v", err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Bookmark or folder not found",
			})
		}
		if err != nil {
			log.Printf("Failed to edit bookmark: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Failed to edit bookmark",
			})
		}

		// Bookmark response type
		type BookmarkResponse struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			Tags      string `json:"tags"`
			FolderID  *int64 `json:"folder_id,omitempty"`
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
		bookmarkID, err := strconv.ParseInt(c.FormValue("bookmark_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid bookmark ID",
			})
		}

		err = database.DeleteBookmark(userID, bookmarkID)
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to delete bookmark %d: %v", bookmarkID, err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Bookmark not found",
			})
		}
		if err != nil {
			log.Printf("Failed to delete bookmark %d: %v", bookmarkID, err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Something went wrong",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}))

	e.POST("/bookmarks/add-folder", auth.RequireAuth(func(c echo.Context) error {
		userID := c.Get("user_id").(int)
		var parentFolderID *int64

		fields, err := readFolderFields(c)
		if err != nil {
			log.Printf("Invalid folder fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}

		if c.FormValue("parent_folder_id") == "null" || c.FormValue("parent_folder_id") == "" {
			parentFolderID = nil
		} else {
			parentFolderID = new(int64)
			*parentFolderID, err = strconv.ParseInt(c.FormValue("parent_folder_id"), 10, 64)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"error":   "Invalid parent folder ID",
				})
			}
		}

		var public bool = c.FormValue("public") == "true"

		newFolder, err := database.AddBookmarkFolder(userID, fields.Name, parentFolderID, public)
		if errors.Is(err, database.ErrDuplicateName) {
			log.Printf("Failed to add folder: %v", err)
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"success": false,
				"error":   "There is already a folder by that name in that place",
			})
		}
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to add folder: %v", err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Folder not found",
			})
		}
		if err != nil {
			log.Printf("Failed to add folder: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Failed to add folder",
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
		folderID, err := strconv.ParseInt(c.FormValue("folder_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid folder ID",
			})
		}

		fields, err := readFolderFields(c)
		if err != nil {
			log.Printf("Invalid folder fields: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}

		newFolder, err := database.RenameBookmarkFolder(userID, folderID, fields.Name)
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("Failed to rename folder %d: %v", folderID, err)
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"error":   "Folder not found",
			})
		} else if errors.Is(err, database.ErrDuplicateName) {
			log.Printf("Failed to rename folder %d: %v", folderID, err)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "A folder of that name already exists there",
			})
		} else if err != nil {
			log.Printf("Failed to rename folder %d: %v", folderID, err)
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"error":   "Something went wrong",
			})
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
		folderID, err := strconv.ParseInt(c.FormValue("folder_id"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error":   "Invalid folder ID",
			})
		}

		err = database.DeleteBookmarkFolder(userID, folderID)
		if errors.Is(err, database.ErrNotFound) {
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

	e.POST("/login", func(c echo.Context) error {
		username := c.FormValue("username")
		err := auth.Login(c, username, c.FormValue("password"))
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    loginPage.Title,
					"NavItems": navFor(false),
					"Username": username,
					"Error":    "Invalid username or password",
				})
			}

			log.Printf("Failed to authenticate user: %v", err)
			return c.Render(http.StatusInternalServerError, "login.html", map[string]interface{}{
				"BasePath": basePath,
				"Title":    loginPage.Title,
				"NavItems": navFor(false),
				"Username": username,
				"Error":    "Something went wrong. Please wait a few minutes and try again.",
			})
		}

		basePath := c.Get("basePath").(string)
		var homePath string = basePath + "/"

		return c.Redirect(http.StatusSeeOther, homePath)
	}, middleware.RateLimiter(
		middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate: 1.0 / 12, Burst: 5, ExpiresIn: 10 * time.Minute,
			},
		)))

	e.POST("/logout", auth.Logout)

	// Blacklisted template names
	var blacklist []string = []string{"header", "footer", "error", "home", "bookmarks"}

	// Route all other templates
	for _, page := range pages {
		if slices.Contains(blacklist, page.Name) {
			continue
		}

		if page.RequireAuth {
			e.GET(page.URL, auth.RequireAuth(func(c echo.Context) error {
				return c.Render(http.StatusOK, page.Name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    page.Title,
					"NavItems": navFor(true),
				})
			}))
		} else if page.HideIfAuth {
			e.GET(page.URL, auth.RequireNoAuth(func(c echo.Context) error {
				return c.Render(http.StatusOK, page.Name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    page.Title,
					"NavItems": navFor(false),
				})
			}))
		} else {
			e.GET(page.URL, func(c echo.Context) error {
				var isAuthenticated bool = auth.IsUserAuthenticated(c)
				return c.Render(http.StatusOK, page.Name+".html", map[string]interface{}{
					"BasePath": basePath,
					"Title":    page.Title,
					"NavItems": navFor(isAuthenticated),
				})
			})
		}
	}
	return nil
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
