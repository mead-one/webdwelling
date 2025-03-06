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

// ### Database initialisation ###
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

// Close the database connection
func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}

// ### Database schema ###
// Create tables if they don't exist
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
        UNIQUE(username, email, subdirectory)
    )`); err != nil {
        log.Fatalf("Failed to create users table: %v", err)
    }

    if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS bookmark_folders (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        parent_folder_id INTEGER,
        public INTEGER NOT NULL DEFAULT 1,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(parent_folder_id) REFERENCES folders(id),
        UNIQUE(user_id, name)
    )`); err != nil {
        log.Fatalf("Failed to create folders table: %v", err)
    }

    if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS bookmarks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        title TEXT,
        url TEXT NOT NULL,
        tags TEXT,
        folder_id INTEGER,
        public INTEGER NOT NULL DEFAULT 1,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(folder_id) REFERENCES folders(id)
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

// ### Database types ###
type User struct {
    ID int
    Username string
    Password string
    Email *string
    Subdirectory *string
    Admin bool
    CreatedAt string
    LastLogin *string
}

type BookmarkFolder struct {
    ID int
    Name string
    ParentFolderID *int
    CreatedAt string
    ChildFolders []*BookmarkFolder
    ChildBookmarks []*Bookmark
}

type Bookmark struct {
    ID int
    Title string
    URL string
    Tags string
    FolderID *int
    Public bool
}

// ### User functions ###
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
        &user.ID, &user.Username, &user.Password, &user.Email, &user.Subdirectory, &user.Admin, &user.CreatedAt, &user.LastLogin,
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

// ### Bookmark functions ###
// Get all bookmarks for a user, returns a nested tree of bookmark folders and bookmarks
func GetBookmarksByUserID(userID int, includePrivate bool) (BookmarkFolder, error) {
    bookmarks := BookmarkFolder{}
    folderMap := make(map[int]*BookmarkFolder)

    // Get bookmark folders
    rows, err := DB.Query("SELECT id,name,parent_folder_id,created_at FROM bookmark_folders WHERE user_id = ?", userID)
    if err != nil {
        return bookmarks, fmt.Errorf("Failed to get bookmark folders: %v", err)
    }

    // Iterate over rows
    for rows.Next() {
        folder := BookmarkFolder{}
        var parentFolderID *int

        if err := rows.Scan(
            &folder.ID, &folder.Name, &parentFolderID, &folder.CreatedAt,
        ); err != nil {
            return bookmarks, fmt.Errorf("Failed to scan bookmark folder: %v", err)
        }

        folder.ParentFolderID = parentFolderID

        folderMap[folder.ID] = &folder
    }
    rows.Close()

    for _, folder := range folderMap {
        if folder.ParentFolderID == nil {
            bookmarks.ChildFolders = append(bookmarks.ChildFolders, folder)
        } else {
            folderMap[*folder.ParentFolderID].ChildFolders = append(folderMap[*folder.ParentFolderID].ChildFolders, folder)
        }
    }

    // Get bookmarks
    rows, err = DB.Query("SELECT id,title,url,tags,folder_id,public FROM bookmarks WHERE user_id = ?", userID)
    if err != nil {
        return bookmarks, fmt.Errorf("Failed to get bookmarks: %v", err)
    }

    // Iterate over rows
    for rows.Next() {
        bookmark := Bookmark{}
        var folderID *int

        if err := rows.Scan(
            &bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.Tags, &folderID, &bookmark.Public,
        ); err != nil {
            return bookmarks, fmt.Errorf("Failed to scan bookmark: %v", err)
        }

        bookmark.FolderID = folderID

        if !bookmark.Public && !includePrivate {
            continue
        } else if bookmark.FolderID == nil {
            bookmarks.ChildBookmarks = append(bookmarks.ChildBookmarks, &bookmark)
        } else {
            folderMap[*bookmark.FolderID].ChildBookmarks = append(folderMap[*bookmark.FolderID].ChildBookmarks, &bookmark)
        }
    }
    rows.Close()
    
    return bookmarks, nil
}

// // Add a new bookmark folder
// func AddBookmarkFolder(userID int, name string, parentFolderID *int, public bool) (*BookmarkFolder, error) {
//     // Check if folder name already exists
//     folder, err := GetBookmarkFolderByName(userID, name)
//     if err != nil {
//         return nil, err
//     }
//     if folder != nil {
//         return nil, fmt.Errorf("Folder name already exists")
//     }
//
//     // Create folder
//     _, err = DB.Exec(
//         "INSERT INTO bookmark_folders (user_id,name,parent_folder_id,public) VALUES (?,?,?,?)",
//         userID, name, parentFolderID, public,
//     )
//     if err != nil {
//         return nil, fmt.Errorf("Failed to create bookmark folder: %v", err)
//     }
//
//     // Create folder struct
//     folder := BookmarkFolder{
//         Name: name,
//         ParentFolderID: parentFolderID,
//         CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
//     }
//     return &folder, nil
// }
//
// // Get a bookmark folder by name
// func GetBookmarkFolderByName(userID int, name string) (*BookmarkFolder, error) {
//     folder := &BookmarkFolder{}
//     err := DB.QueryRow("SELECT * FROM bookmark_folders WHERE user_id = ? AND name = ?", userID, name).Scan(
//         &folder.ID, &folder.Name, &folder.ParentFolderID, &folder.CreatedAt,
//     )
//     if err != nil {
//         return nil, fmt.Errorf("Failed to get bookmark folder by name: %v", err)
//     }
//
//     return folder, nil
// }
//
