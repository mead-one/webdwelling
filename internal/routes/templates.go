// Templating logic for routes
package routes

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Give every template the CSRF token set by the CSRF middleware
	if m, ok := data.(map[string]interface{}); ok {
		m["CSRFToken"] = c.Get("csrf")
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

func TemplateRenderer(templatesDir string) *Template {
	return &Template{
		templates: template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html"))),
	}
}

type NavItem struct {
	Name        string
	URL         string
	Weight      int
	Method      string // Empty means an ordinary GET link
	RequireAuth bool
	HideIfAuth  bool
}

func filterNavItems(items []NavItem, isAuthenticated bool) []NavItem {
	var filtered []NavItem = make([]NavItem, 0, len(items))

	for _, item := range items {
		if item.RequireAuth && !isAuthenticated || item.HideIfAuth && isAuthenticated {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func buildNavItems(files []string) []NavItem {
	var navItems []NavItem
	// Set up language caser for casing titles
	caser := cases.Title(language.BritishEnglish)

	for _, file := range files {
		var base string = filepath.Base(file)
		var name string = strings.TrimSuffix(base, ".html")
		var url string
		if name == "home" {
			url = "/"
		} else {
			url = "/" + name
		}

		// Read comment on first line
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		line, err := getNavComment(f)
		f.Close()
		if err != nil {
			fmt.Println(err)
			continue
		}

		include, weight, requireAuth, hideIfAuth := parseNavComment(line)

		if include {
			navItems = append(navItems, NavItem{
				Name:        caser.String(name),
				URL:         url,
				Weight:      weight,
				RequireAuth: requireAuth,
				HideIfAuth:  hideIfAuth,
			})
		}
	}

	navItems = append(navItems, NavItem{
		Name:        "Logout",
		URL:         "/logout",
		Weight:      100,
		Method:      "POST",
		RequireAuth: true,
		HideIfAuth:  false,
	})

	sort.Slice(navItems, func(i, j int) bool {
		if navItems[i].Weight == navItems[j].Weight {
			return navItems[i].Name < navItems[j].Name
		}
		return navItems[i].Weight < navItems[j].Weight
	})

	return navItems
}

// Scan every line in file until one with nav: comment is found
func getNavComment(f *os.File) (line string, err error) {
	scanner := bufio.NewScanner(f)

	// Values contained in metadata comment
	var keys []string = []string{"nav:", "weight:", "auth:"}

	// Scan until metadata comment is found
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "<!--") && strings.HasSuffix(line, "-->") {
			for _, key := range keys {
				if strings.Contains(line, key) {
					return line, nil
				}
			}
		}
	}

	return "", errors.New("no nav comment found in file: " + f.Name())
}

func parseNavComment(line string) (include bool, weight int, requireAuth bool, hideIfAuth bool) {
	// Default values
	include = false
	weight = 100
	requireAuth = false
	hideIfAuth = false

	// Extract metadata
	if strings.Contains(line, "nav:") {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<!--"), "-->"))
		for len(line) > 0 {
			var part string
			part, line, _ = strings.Cut(line, ",")
			part = strings.TrimSpace(part)

			if strings.HasPrefix(part, "nav:") {
				include = strings.TrimSpace(strings.TrimPrefix(part, "nav:")) == "include"
			} else if strings.HasPrefix(part, "weight:") {
				weight, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(part, "weight:")))
			} else if strings.HasPrefix(part, "auth:") {
				authValue := strings.TrimSpace(strings.TrimPrefix(part, "auth:"))
				if authValue == "require" {
					requireAuth = true
				} else if authValue == "hide" {
					hideIfAuth = true
				}
			}
		}
	}

	return include, weight, requireAuth, hideIfAuth
}
