package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/mead-one/webdwelling/config"
	"github.com/mead-one/webdwelling/internal/database"
)

var (
	configPath string
	username   string
	password   string
	email      string
	publicSlug string
	admin      bool
)

func init() {
	flag.StringVar(&configPath, "config", config.DefaultPath, "path to the config file")
	flag.StringVar(&username, "username", "", "New user's username")
	flag.StringVar(&email, "email", "", "New user's email address")
	flag.StringVar(&publicSlug, "public_slug", "", "New user's public slug, used in their public URL")
	flag.BoolVar(&admin, "admin", false, "Whether the new user has admin privileges")
}

func main() {
	flag.Parse()
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	publicSlug = strings.ReplaceAll(publicSlug, " ", "")
	reader := bufio.NewReader(os.Stdin)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// --- Validation ---
	if username == "" || !isValidEmail(email) || publicSlug == "" {
		var err error

		if username == "" {
			fmt.Println("Username:")
			fmt.Print("-> ")
			username, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Failed to read a valid username")
				return
			}
			username = strings.TrimSpace(username)
			if len(username) < 3 || len(username) > 12 {
				fmt.Println("Username must be between 3 and 12 characters long")
				return
			}
		}

		if !isValidEmail(email) {
			if email != "" {
				fmt.Println("Invalid email address provided.")
			}

			fmt.Println("Email address:")
			fmt.Print("-> ")
			email, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Failed to read a valid email address")
				return
			}
			email = strings.TrimSpace(email)
			if !isValidEmail(email) {
				fmt.Println("Invalid email address")
				return
			}
		}

		if publicSlug == "" {
			fmt.Println("Public slug (for public page URL):")
			fmt.Print("-> ")
			publicSlug, err = reader.ReadString('\n')
			if err != nil {
				fmt.Println("Failed to read a valid public slug")
				return
			}
			publicSlug = strings.TrimSpace(publicSlug)
			if publicSlug == "" || len(publicSlug) > 12 {
				fmt.Println("Public slugs must be 1-12 characters long")
			}
		}
	}

	fmt.Println("Password:")
	fmt.Print("-> ")
	password, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read a valid password")
		return
	}
	password = strings.TrimSpace(password)

	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Public slug: %s\n", publicSlug)
	fmt.Printf("User is admin: %t\n", admin)

	// --- Database Operations ---
	database.InitDB(cfg.DataDirectory)

	user, err := database.CreateUser(username, password, &email, &publicSlug, admin)
	if err != nil {
		fmt.Printf("Failed to create user: %v\n", err)
		return
	}

	fmt.Printf("Registered user %s (%s)\n", user.Username, *user.Email)

	database.CloseDB()
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
