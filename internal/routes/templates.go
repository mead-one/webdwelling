// Templating logic for routes
package routes

import (
    "bufio"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"

    "golang.org/x/text/cases"
    "golang.org/x/text/language"
)

type NavItem struct {
    Name string
    URL string
    Weight int
}

func GetNavItems() []NavItem {
    var pages []NavItem
    files, _ := filepath.Glob("templates/*.html")

    for _, file := range files {
        base := filepath.Base(file)
        name := strings.TrimSuffix(base, ".html")

        // Exclude the base template
        if name == "base" {
            continue
        }

        // Default values
        include := true
        weight := 100

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
                parts := strings.Split(line, ",")
                for _, part := range parts {
                    part = strings.TrimSpace(strings.Trim(part, "<!-- -->"))
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
