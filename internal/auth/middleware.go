package auth

import (
    "net/http"
    "time"
    "errors"

    "github.com/golang-jwt/jwt/v4"
    "github.com/labstack/echo/v4"
    "github.com/mead-one/webdwelling/internal/database"
)

// Secret used to sign JWT tokens
// TODO: Move to config file or environment variable
var JWTSecret = []byte("secret")

// Custom claims for JWT tokens
type JWTCustomClaims struct {
    UserID int `json:"user_id"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}

// Authenticate a user and assign a cookie with their JWT token
func Login(c echo.Context) error {
    username := c.FormValue("username")
    password := c.FormValue("password")

    user, err := database.AuthenticateUser(username, password)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user")
    }
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "Invalid username or password")
    }

    // Create claims
    claims := &JWTCustomClaims{
        user.ID,
        user.Username,
        jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 14)),
        },
    }

    // Create JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    // Encode token
    tk, err := token.SignedString(JWTSecret)
    if err != nil {
        return err
    }

    // Set cookie
    cookie := new(http.Cookie)
    cookie.Name = "jwt"
    cookie.Value = tk
    cookie.Expires = time.Now().Add(time.Hour * 24 * 14)
    cookie.Path = "/"
    cookie.HttpOnly = true
    c.SetCookie(cookie)

    return c.Redirect(http.StatusSeeOther, "/")
}

func IsUserAuthenticated(c echo.Context) bool {
    err := checkCookie(c)
    if err != nil {
        return false
    }

    return true
}

// Check if a user is logged in
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        err := checkCookie(c)
        if err != nil {
            return c.Redirect(http.StatusSeeOther, "/login")
        }

        return next(c)
    }
}

// Check if a user is not logged in
func RequireNoAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        err := checkCookie(c)
        if err == nil {
            return c.Redirect(http.StatusSeeOther, "/")
        }

        return next(c)
    }
}

func checkCookie(c echo.Context) error {
    cookie, err := c.Cookie("jwt")
    if err != nil {
        return err
    }

    // Parse JWT token
    token, err:= jwt.ParseWithClaims(cookie.Value, &JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        return JWTSecret, nil
    })

    if err != nil {
        return err
    }

    if !token.Valid {
        // Clear invalid cookie
        cookie := new(http.Cookie)
        cookie.Name = "jwt"
        cookie.Value = ""
        cookie.Expires = time.Now().Add(-time.Hour)
        cookie.Path = "/"
        cookie.HttpOnly = true
        c.SetCookie(cookie)

        return errors.New("Invalid JWT token")
    }
    
    // Get claims
    claims, ok := token.Claims.(*JWTCustomClaims)
    if !ok {
        return errors.New("Invalid JWT claims")
    }

    // Set user info in context
    c.Set("user_id", claims.UserID)
    c.Set("username", claims.Username)

    return nil
}

func Logout(c echo.Context) error {
    cookie := new(http.Cookie)
    cookie.Name = "jwt"
    cookie.Value = ""
    cookie.Expires = time.Now().Add(-time.Hour)
    cookie.Path = "/"
    cookie.HttpOnly = true
    c.SetCookie(cookie)

    return c.Redirect(http.StatusSeeOther, "/login")
}

