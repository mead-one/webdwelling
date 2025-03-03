// Templating logic for routes
package routes

import (
    "io"
    "html/template"
    "bufio"
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
    return t.templates.ExecuteTemplate(w, name, data)
}

func TemplateRenderer(templatesDir string) *Template {
    return &Template{
        templates: template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html"))),
    }
}

type NavItem struct {
    Name string
    URL string
    Weight int
}

func GetNavItems() []NavItem {
    var pages []NavItem
    files, _ := filepath.Glob(filepath.Join("web", "templates", "*.html"))

    for _, file := range files {
        base := filepath.Base(file)
        name := strings.TrimSuffix(base, ".html")

        // Exclude the base template
        if name == "header" || name == "footer" {
            continue
        }

        // Default values
        var include bool = true
        var weight int = 100

        // Read comment on first line
        f, err := os.Open(file)
        if err != nil {
            continue
        }
        defer f.Close()

        scanner := bufio.NewScanner(f)
        if scanner.Scan() {
            line := scanner.Text()

            // Extract metadata
            if strings.Contains(line, "nav:") {
                line = strings.TrimSpace(strings.Trim(line, "<!-- -->"))
                for len(line) > 0 {
                    var part string
                    part, line, _ = strings.Cut(line, ",")
                    if strings.HasPrefix(part, "nav:") {
                        include = strings.TrimSpace(strings.TrimPrefix(part, "nav:")) == "include"
                    } else if strings.HasPrefix(part, "weight:") {
                        weight, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(part, "weight:")))
                    }
                }
            }
        }

        if !include {
            continue
        }

        // Set up language caser for casing titles
        caser := cases.Title(language.BritishEnglish)

        pages = append(pages, NavItem{Name: caser.String(name), URL: "/" + name, Weight: weight})

        sort.Slice(pages, func(i, j int) bool {
            if pages[i].Weight == pages[j].Weight {
                return pages[i].Name < pages [j].Name
            }
            return pages[i].Weight < pages[j].Weight
        })
    }

    return pages
}
