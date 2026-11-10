package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/mead-one/webdwelling/config"
	"github.com/mead-one/webdwelling/internal/auth"
	"github.com/mead-one/webdwelling/internal/database"
	"github.com/mead-one/webdwelling/internal/routes"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "path to the config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	auth.JWTSecret = cfg.JWTSecret
	auth.SecureCookies = cfg.SecureCookies

	e := echo.New()
	// Prevent spoofing IP addresses using X-Forwarded-For header
	e.IPExtractor = echo.ExtractIPFromXFFHeader()

	// Maximum 16KB request bodies
	e.Use(middleware.BodyLimitWithConfig(middleware.BodyLimitConfig{
		Limit: "16K",
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/bookmarks/import-bookmarks"
		},
	}))
	// Protection against XSS, content type sniffing, clickjacking, etc.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "default-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'",
		// Don't tell bookmarked sites the address of this app
		ReferrerPolicy: "same-origin",
	}))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("basePath", cfg.BasePath)
			return next(c)
		}
	})

	database.InitDB(cfg.DataDirectory)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Reject POSTs that don't send back the token stored in the _csrf cookie
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token,form:_csrf",
		CookiePath:     "/",
		CookieMaxAge:   int(auth.SessionLength.Seconds()),
		CookieSecure:   cfg.SecureCookies,
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
	}))

	// Register routes
	err = routes.RegisterRoutes(e, cfg.BasePath, cfg.TemplatesDirectory, cfg.StaticDirectory)
	if err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Port)))
}
