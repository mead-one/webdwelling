// Routes contains all of the routes for webdwelling
package routes

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

func Index(c echo.Context) error {
    return c.String(http.StatusOK, "Hello, World!")
}

func Bookmarks(c echo.Context) error {
    return c.String(http.StatusOK, "Bookmarks")
}

func CreateBookmark(c echo.Context) error {
    title := c.FormValue("title");
    url := c.FormValue("url");

    // TODO: Return to /bookmarks, with success message
    return c.String(http.StatusOK, "Create Bookmark: " + title + " (" + url + ")")
}

