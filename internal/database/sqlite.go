package database

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "time"
    "path/filepath"

    _ "github.com/mattn/go-sqlite3"
    "golang.org/x/crypto/bcrypt"
)

// Database connection
var DB *sql.DB

// Initialise the database connection
func InitDB() {
    // Ensure data directory exists
    dataDir := "./data"
    if _, err := os.Stat(dataDir); os.IsNotExist(err) {
        if err := os.Mkdir(dataDir, 0755); err != nil {
            log.Fatalf("Failed to create data directory: %v", err)
        }
    }

    dbPath := filepath.Join(dataDir, "webdwelling.db")
    db, err:= sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }

    // Test connection
    if err = db.Ping(); err != nil {
        log.Fatalf("Failed to ping database: %v", err)
    }

    DB = db

    // Create tables if they don't exist
    createTables()
}

func createTables() {
    // Create table if it doesn't exist
    if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL,
        password TEXT NOT NULL,
        email TEXT,
        subdirectory TEXT,
        admin BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        last_login TEXT,
        UNIQUE(username),
        UNIQUE(email),
        UNIQUE(subdirectory)
    )`); err != nil {
        log.Fatalf("Failed to create users table: %v", err)
    }

    if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS bookmarks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        title TEXT,
        url TEXT NOT NULL,
        path TEXT,
        public INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY(user_id) REFERENCES users(id)
    )`); err != nil {
        log.Fatalf("Failed to create bookmarks table: %v", err)
    }

    // Create trigger to ensure no clash between users' usernames and subdirectories
    if _, err := DB.Exec(`CREATE TRIGGER IF NOT EXISTS prevent_username_subdirectory_clash
    BEFORE INSERT ON users
    FOR EACH ROW
    BEGIN
        SELECT RAISE(ABORT, 'Username taken by existing subdirectory')
        FROM users WHERE subdirectory = NEW.username;
        
        SELECT RAISE(ABORT, 'Subdirectory taken by existing username')
        FROM users WHERE username = NEW.subdirectory;
    END
    `); err != nil {
        log.Fatalf("Failed to create prevent_username_subdirectory_clash trigger: %v", err)
    }
}

type User struct {
    ID int
    Username string
    Password string
    Email *string
    Subdirectory *string
    Admin bool
    CreatedAt string
}

func CreateUser(username string, password string, email *string, subdirectory *string, admin bool) (*User, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("Failed to hash password: %v", err)
    }

    // Write user to database
    _, err = DB.Exec(
        "INSERT INTO users (username,password,email,subdirectory,admin) VALUES (?,?,?,?,?)",
        username, string(hashedPassword), email, subdirectory, admin,
    )
    if err != nil {
        return nil, fmt.Errorf("Failed to write user to database: %v", err)
    }

    // Create user struct
    user := User{
        Username: username,
        Password: string(hashedPassword),
        Email: email,
        Subdirectory: subdirectory,
        Admin: admin,
        CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
    }
    return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
    user := &User{}
    err := DB.QueryRow("SELECT * FROM users WHERE username = ?", username).Scan(
        &user.ID, &user.Username, &user.Password, &user.Email, &user.Subdirectory, &user.Admin, &user.CreatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("Failed to get user by username: %v", err)
    }

    return user, nil
}

func AuthenticateUser(username string, password string) (*User, error) {
    user, err := GetUserByUsername(username)
    if err != nil {
        return nil, err
    }
    if user == nil {
        // User not found
        return nil, nil
    }

    // Compare passwords
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        // Passwords don't match
        return nil, nil
    }

    // Update last login time
    _, err = DB.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE username = ?", username)
    if err != nil {
        log.Printf("Failed to update last login time: %v", err)
    }

    return user, nil
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}

