package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/mead-one/webdwelling/internal/database"
)

// Secret used to sign JWT tokens, set from config at startup
var JWTSecret []byte

// Whether cookies are marked Secure (HTTPS only), set from config at startup
var SecureCookies bool

var ErrInvalidCredentials = errors.New("invalid username or password")

// How long a login lasts before the user must log in again
const SessionLength = 14 * 24 * time.Hour

// Name of the cookie holding the user's JWT token
const authCookieName = "jwt"

// Custom claims for JWT tokens
type JWTCustomClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Authenticate a user and assign a cookie with their JWT token
func Login(c echo.Context, username, password string) error {
	user, err := database.AuthenticateUser(username, password)
	if err != nil {
		return fmt.Errorf("authenticate %s: %w", username, err)
	}
	if user == nil {
		return ErrInvalidCredentials
	}

	// Create claims
	claims := &JWTCustomClaims{
		user.ID,
		user.Username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(SessionLength)),
		},
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Encode token
	tk, err := token.SignedString(JWTSecret)
	if err != nil {
		return err
	}

	setAuthCookie(c, tk)

	return nil
}

// Build the auth cookie, so setting and clearing it always use the same attributes
func newAuthCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

// Give the user a cookie holding their JWT token
func setAuthCookie(c echo.Context, token string) {
	c.SetCookie(newAuthCookie(token, int(SessionLength.Seconds())))
}

// Delete the user's auth cookie
func clearAuthCookie(c echo.Context) {
	// A negative MaxAge tells the browser to delete the cookie now
	c.SetCookie(newAuthCookie("", -1))
}

func IsUserAuthenticated(c echo.Context) bool {
	err := checkCookie(c)
	return err == nil
}

// Check if a user is logged in
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := checkCookie(c)
		if err != nil {
			var loginPath string = c.Get("basePath").(string) + "/login"
			return c.Redirect(http.StatusSeeOther, loginPath)
		}

		return next(c)
	}
}

// Check if a user is not logged in
func RequireNoAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := checkCookie(c)
		if err == nil {
			var homePath string = c.Get("basePath").(string) + "/"
			return c.Redirect(http.StatusSeeOther, homePath)
		}

		return next(c)
	}
}

func checkCookie(c echo.Context) error {
	cookie, err := c.Cookie(authCookieName)
	if err != nil {
		return err
	}

	// Parse JWT token
	token, err := jwt.ParseWithClaims(cookie.Value, &JWTCustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	// ParseWithClaims returns an error for any invalid token (bad signature, expired, etc.)
	if err != nil || !token.Valid {
		clearAuthCookie(c)
		return errors.New("invalid JWT token")
	}

	// Get claims
	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok {
		return errors.New("invalid JWT claims")
	}

	// Set user info in context
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)

	return nil
}

func Logout(c echo.Context) error {
	clearAuthCookie(c)

	var loginPath string = c.Get("basePath").(string) + "/login"

	return c.Redirect(http.StatusSeeOther, loginPath)
}
