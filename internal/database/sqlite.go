package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// ### Database initialisation ###
// Database connection
var DB *sql.DB

// A dummy hash to check passwords against, so attempted logins to non-existent
// users take as long to process as real users to mitigate against brute force
// attacks
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("b7$MQ4lIIp"), bcrypt.DefaultCost)

// Initialise the database connection
func InitDB() {
	// Ensure data directory exists
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}

	dbPath := filepath.Join(dataDir, "webdwelling.db?_foreign_keys=on")
	db, err := sql.Open("sqlite3", dbPath)
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
	var err error

	if _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL,
        email TEXT UNIQUE,
        public_slug TEXT UNIQUE NOT NULL,
        admin BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
        last_login TEXT
    )`); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	if _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS bookmark_folders (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        parent_folder_id INTEGER,
        public INTEGER NOT NULL DEFAULT 1,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(parent_folder_id) REFERENCES bookmark_folders(id) ON DELETE CASCADE,
        UNIQUE(user_id, parent_folder_id, name)
    )`); err != nil {
		log.Fatalf("Failed to create bookmark_folders table: %v", err)
	}
	if _, err = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS bookmark_folders_unique_root_name
	ON bookmark_folders(user_id, name) WHERE parent_folder_id IS NULL`); err != nil {
		log.Fatalf("Failed to create unique index on top-level folders: %v", err)
	}

	if _, err = DB.Exec(`CREATE TABLE IF NOT EXISTS bookmarks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        title TEXT,
        url TEXT NOT NULL,
        tags TEXT,
        folder_id INTEGER,
        public INTEGER NOT NULL DEFAULT 1,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY(user_id) REFERENCES users(id),
        FOREIGN KEY(folder_id) REFERENCES bookmark_folders(id) ON DELETE CASCADE
    )`); err != nil {
		log.Fatalf("Failed to create bookmarks table: %v", err)
	}
}

// ### Database types ###
type User struct {
	ID         int
	Username   string
	Password   string
	Email      *string
	PublicSlug *string
	Admin      bool
	CreatedAt  string
	LastLogin  *string
}

type BookmarkFolder struct {
	ID             int
	Name           string
	ParentFolderID *int
	Public         bool
	CreatedAt      string
	ChildFolders   []*BookmarkFolder
	ChildBookmarks []*Bookmark
}

type Bookmark struct {
	ID        int
	Title     string
	URL       string
	Tags      string
	FolderID  *int
	Public    bool
	CreatedAt string
}

// ### User functions ###
func CreateUser(username string, password string, email *string, public_slug *string, admin bool) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Write user to database
	_, err = DB.Exec(`
	INSERT INTO users (username,password,email,public_slug,admin)
	VALUES (:username, :password, :email, :public_slug, :admin)
	`, sql.Named("username", username), sql.Named("password", string(hashedPassword)),
		sql.Named("email", email), sql.Named("public_slug", public_slug),
		sql.Named("admin", admin))

	if err != nil {
		return nil, fmt.Errorf("failed to write user to database: %w", err)
	}

	// Create user struct
	user := User{
		Username:   username,
		Password:   string(hashedPassword),
		Email:      email,
		PublicSlug: public_slug,
		Admin:      admin,
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}
	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := DB.QueryRow(`
	SELECT id,username,password,email,public_slug,admin,created_at,last_login
	FROM users
	WHERE username = :username`, sql.Named("username", username)).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &user.PublicSlug, &user.Admin, &user.CreatedAt, &user.LastLogin,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

func AuthenticateUser(username string, password string) (*User, error) {
	user, err := GetUserByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// Passwords don't match
		return nil, nil
	}

	// Update last login time
	_, err = DB.Exec(`
	UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE username = :username`, sql.Named("username", username))
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
	var folders []*BookmarkFolder

	// Get bookmark folders
	rows, err := DB.Query(`
	SELECT id,name,parent_folder_id,public,created_at
	FROM bookmark_folders
	WHERE user_id = :user_id AND (public = true OR :include_private)
	ORDER BY name COLLATE NOCASE
	`, sql.Named("user_id", userID), sql.Named("include_private", includePrivate),
	)
	if err != nil {
		return bookmarks, fmt.Errorf("failed to get bookmark folders: %w", err)
	}
	defer rows.Close()

	// Iterate over rows
	for rows.Next() {
		folder := BookmarkFolder{}
		var parentFolderID *int

		if err := rows.Scan(
			&folder.ID, &folder.Name, &parentFolderID, &folder.Public, &folder.CreatedAt,
		); err != nil {
			return bookmarks, fmt.Errorf("failed to scan bookmark folder: %w", err)
		}

		folder.ParentFolderID = parentFolderID

		folderMap[folder.ID] = &folder
		folders = append(folders, &folder)
	}
	if err := rows.Err(); err != nil {
		return bookmarks, fmt.Errorf("failed to scan bookmark folder: %w", err)
	}

	for _, folder := range folders {
		parent := &bookmarks
		if folder.ParentFolderID != nil {
			p, ok := folderMap[*folder.ParentFolderID]
			if !ok {
				continue // Parent is private or missing
			}
			parent = p
		}
		parent.ChildFolders = append(parent.ChildFolders, folder)
	}

	// Get bookmarks
	rows, err = DB.Query(`
	SELECT id,title,url,tags,folder_id,public
	FROM bookmarks
	WHERE user_id = :user_id AND (public = true OR :include_private)
	ORDER BY title COLLATE NOCASE
	`, sql.Named("user_id", userID), sql.Named("include_private", includePrivate))
	if err != nil {
		return bookmarks, fmt.Errorf("failed to get bookmarks: %w", err)
	}
	defer rows.Close()

	// Iterate over rows
	for rows.Next() {
		bookmark := Bookmark{}
		var folderID *int

		if err := rows.Scan(
			&bookmark.ID, &bookmark.Title, &bookmark.URL, &bookmark.Tags, &folderID, &bookmark.Public,
		); err != nil {
			return bookmarks, fmt.Errorf("failed to scan bookmark: %w", err)
		}

		bookmark.FolderID = folderID

		parent := &bookmarks
		if bookmark.FolderID != nil {
			f, ok := folderMap[*bookmark.FolderID]
			if !ok {
				continue // Folder is private or missing
			}
			parent = f
		}
		parent.ChildBookmarks = append(parent.ChildBookmarks, &bookmark)
	}
	if err := rows.Err(); err != nil {
		return bookmarks, fmt.Errorf("failed to scan bookmark: %w", err)
	}

	return bookmarks, nil
}

// Add a new bookmark
func AddBookmark(userID int, title string, url string, tags string, folderID *int, public bool) (*Bookmark, error) {
	lastID := 0

	// Write bookmark to database
	err := DB.QueryRow(`
		INSERT INTO bookmarks (user_id,title,url,tags,folder_id,public)
		SELECT :user_id, :title, :url, :tags, :folder_id, :public
		WHERE :folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders WHERE id = :folder_id AND user_id = :user_id)
		RETURNING id`,
		sql.Named("user_id", userID), sql.Named("title", title), sql.Named("url", url),
		sql.Named("tags", tags), sql.Named("folder_id", folderID), sql.Named("public", public),
	).Scan(&lastID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("folder_id is not a valid folder")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create bookmark: %w", err)
	}

	// Create bookmark struct
	bookmark := &Bookmark{
		ID:        lastID,
		Title:     title,
		URL:       url,
		Tags:      tags,
		FolderID:  folderID,
		Public:    public,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	return bookmark, nil
}

// Move a bookmark to a folder
func MoveBookmark(userID int, bookmarkID int, folderID *int) error {
	// Check if the user is the owner of the bookmark
	bookmarkUserID := 0
	err := DB.QueryRow(`
	SELECT user_id FROM bookmarks WHERE id = :bookmark_id
	`, sql.Named("bookmark_id", bookmarkID)).Scan(&bookmarkUserID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark user ID: %w", err)
	}
	if bookmarkUserID != userID {
		return fmt.Errorf("user is not authorized to move this bookmark")
	}

	if folderID != nil {
		// Check if the user is the owner of the folder
		folderUserID := 0
		err = DB.QueryRow(`
		SELECT user_id FROM bookmark_folders WHERE id = :folder_id
		`, sql.Named("folder_id", folderID)).Scan(&folderUserID)
		if err != nil {
			return fmt.Errorf("failed to get folder user ID: %w", err)
		}
		if folderUserID != userID {
			return fmt.Errorf("user is not authorized to move this bookmark")
		}
	}

	// Write bookmark to database
	_, err = DB.Exec(`
	UPDATE bookmarks SET folder_id = :folder_id WHERE id = :bookmark_id
	`, sql.Named("folder_id", folderID), sql.Named("bookmark_id", bookmarkID),
	)
	if err != nil {
		return fmt.Errorf("failed to move bookmark: %w", err)
	}

	return nil
}

// Edit a bookmark
func EditBookmark(userID int, bookmarkID int, title string, url string, tags string, folderID *int, public bool) (*Bookmark, error) {
	var lastID int
	err := DB.QueryRow(`
		UPDATE bookmarks SET title = :title, url = :url, tags = :tags, folder_id = :folder_id, public = :public
		WHERE id = :bookmark_id AND user_id = :user_id
		AND (:folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders WHERE id = :folder_id AND user_id = :user_id))
		RETURNING id
	`, sql.Named("user_id", userID), sql.Named("bookmark_id", bookmarkID),
		sql.Named("title", title), sql.Named("url", url),
		sql.Named("tags", tags), sql.Named("folder_id", folderID),
		sql.Named("public", public)).Scan(&lastID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("bookmark or folder not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to edit bookmark: %w", err)
	}

	// Create bookmark struct
	bookmark := &Bookmark{
		ID:        bookmarkID,
		Title:     title,
		URL:       url,
		Tags:      tags,
		FolderID:  folderID,
		Public:    public,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	return bookmark, nil
}

// Delete a bookmark
func DeleteBookmark(userID int, bookmarkID int) error {
	bookmarkUserID := 0
	err := DB.QueryRow(`
	SELECT user_id FROM bookmarks WHERE id = :bookmark_id
	`, sql.Named("bookmark_id", bookmarkID)).Scan(&bookmarkUserID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark user ID: %w", err)
	}
	if bookmarkUserID != userID {
		return fmt.Errorf("user is not authorized to delete this bookmark")
	}

	_, err = DB.Exec(`
	DELETE FROM bookmarks WHERE id = :bookmark_id
	`, sql.Named("bookmark_id", bookmarkID))
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	return nil
}

// Add a new bookmark folder
func AddBookmarkFolder(userID int, name string, parentFolderID *int, public bool) (*BookmarkFolder, error) {
	lastID := 0

	if parentFolderID != nil {
		// Check if the user is the owner of the parent folder
		var parentFolderUserID int
		err := DB.QueryRow(`
		SELECT user_id FROM bookmark_folders WHERE id = :parent_folder_id
		`, sql.Named("parent_folder_id", parentFolderID)).Scan(&parentFolderUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent folder user ID: %w", err)
		}

		if parentFolderUserID != userID {
			return nil, fmt.Errorf("user is not authorized to create a folder in this folder")
		}
	}

	// Create folder
	err := DB.QueryRow(`
		INSERT INTO bookmark_folders (user_id,name,parent_folder_id,public) VALUES (:user_id, :name, :parent_folder_id, :public) RETURNING id
		`, sql.Named("user_id", userID), sql.Named("name", name), sql.Named("parent_folder_id", parentFolderID), sql.Named("public", public),
	).Scan(&lastID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bookmark folder: %w", err)
	}

	// // Create folder struct
	folder := &BookmarkFolder{
		ID:             lastID,
		Name:           name,
		ParentFolderID: parentFolderID,
		Public:         public,
		CreatedAt:      time.Now().Format("2006-01-02 15:04:05"),
		ChildBookmarks: make([]*Bookmark, 0),
		ChildFolders:   make([]*BookmarkFolder, 0),
	}
	return folder, nil
}

// Move a bookmark folder to a new parent folder
func MoveBookmarkFolder(userID int, folderID int, destinationFolderID *int) error {
	// Check if the destination folder is a descendant of the moved folder
	if FolderIsDescendant(folderID, destinationFolderID) {
		return fmt.Errorf("cannot move a folder into one of its descendants")
	}

	// Check if the user is the owner of the folder
	var storedFolderID int
	err := DB.QueryRow(`
	SELECT user_id FROM bookmark_folders WHERE id = :folder_id
	`, sql.Named("folder_id", folderID)).Scan(&storedFolderID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark folder user ID: %w", err)
	}
	if storedFolderID != userID {
		return fmt.Errorf("user is not authorized to move this folder")
	}

	// Check if the user is the owner of the destination folder
	if destinationFolderID != nil {
		var storedParentFolderUserID *int
		err = DB.QueryRow(`
		SELECT user_id FROM bookmark_folders WHERE id = :destination_folder_id
		`, sql.Named("destination_folder_id", *destinationFolderID)).Scan(&storedParentFolderUserID)
		if err != nil {
			return fmt.Errorf("failed to get parent folder user ID: %w", err)
		}
		if storedParentFolderUserID != nil && *storedParentFolderUserID != userID {
			return fmt.Errorf("user is not authorized to move this folder")
		}
	}

	// Write folder to database
	_, err = DB.Exec(`
	UPDATE bookmark_folders SET parent_folder_id = :destination_folder_id WHERE id = :folder_id
		`, sql.Named("destination_folder_id", destinationFolderID), sql.Named("folder_id", folderID),
	)
	if err != nil {
		return fmt.Errorf("failed to move bookmark folder: %w", err)
	}

	return nil
}

// Check if a folder is a descendant of another folder
func FolderIsDescendant(folderID int, destinationFolderID *int) bool {
	if destinationFolderID == nil {
		return false
	}

	destinationFolder, err := GetBookmarkFolderByID(*destinationFolderID)
	if err != nil {
		return false
	}

	if folderID == destinationFolder.ID {
		return true
	}

	if destinationFolder.ParentFolderID == nil {
		return false
	}

	return FolderIsDescendant(folderID, destinationFolder.ParentFolderID)
}

func RenameBookmarkFolder(userID int, folderID int, name string) (*BookmarkFolder, error) {
	folderUserID := 0
	err := DB.QueryRow(`
	SELECT user_id FROM bookmark_folders WHERE id = :folder_id
	`, sql.Named("folder_id", folderID)).Scan(&folderUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmark folder user ID: %w", err)
	}
	if folderUserID != userID {
		return nil, fmt.Errorf("user is not authorized to rename this folder")
	}

	folder := &BookmarkFolder{}

	err = DB.QueryRow(`
	UPDATE bookmark_folders SET name = :name WHERE id = :folder_id RETURNING id,name,parent_folder_id,public,created_at
		`, sql.Named("name", name), sql.Named("folder_id", folderID),
	).Scan(&folder.ID, &folder.Name, &folder.ParentFolderID, &folder.Public, &folder.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to rename bookmark folder: %w", err)
	}

	return folder, nil
}

func DeleteBookmarkFolder(userID int, folderID int) error {
	folderUserID := 0
	err := DB.QueryRow(`
	SELECT user_id FROM bookmark_folders WHERE id = :folder_id
	`, sql.Named("folder_id", folderID)).Scan(&folderUserID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark folder user ID: %w", err)
	}
	if folderUserID != userID {
		return fmt.Errorf("user is not authorized to delete this folder")
	}

	// Delete all bookmarks in this folder
	_, err = DB.Exec(`
	DELETE FROM bookmarks WHERE folder_id = :folder_id
	`, sql.Named("folder_id", folderID))
	if err != nil {
		return fmt.Errorf("failed to delete bookmarks: %w", err)
	}

	// Recursively delete all descendant folders
	// Get all child folders
	rows, err := DB.Query(`
	SELECT id FROM bookmark_folders WHERE parent_folder_id = :folder_id
	`, sql.Named("folder_id", folderID))
	if err != nil {
		return fmt.Errorf("failed to get child folders: %w", err)
	}
	defer rows.Close()

	// Iterate over rows, storing child folder IDs
	var childFolderIDs []int
	for rows.Next() {
		var childFolderID int
		if err := rows.Scan(&childFolderID); err != nil {
			return fmt.Errorf("failed to scan child folder ID: %w", err)
		}

		childFolderIDs = append(childFolderIDs, childFolderID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to scan child folder ID: %w", err)
	}

	for _, childFolderID := range childFolderIDs {
		// Delete child bookmarks
		_, err = DB.Exec(`
		DELETE FROM bookmarks WHERE folder_id = :child_folder_id
		`, sql.Named("child_folder_id", childFolderID))
		if err != nil {
			return fmt.Errorf("failed to delete child bookmarks: %w", err)
		}

		// Recursively delete child folders
		err = DeleteBookmarkFolder(userID, childFolderID)
		if err != nil {
			return fmt.Errorf("failed to delete child folder: %w", err)
		}
	}

	// Delete the folder
	_, err = DB.Exec(`
	DELETE FROM bookmark_folders WHERE id = :folder_id
	`, sql.Named("folder_id", folderID))
	if err != nil {
		return fmt.Errorf("failed to delete bookmark folder: %w", err)
	}

	return nil
}

// Get a bookmark folder by name
func GetBookmarkFolderByID(folderID int) (*BookmarkFolder, error) {
	folder := &BookmarkFolder{}
	err := DB.QueryRow(`
	SELECT id,name,parent_folder_id,created_at FROM bookmark_folders WHERE id = :folder_id
	`, sql.Named("folder_id", folderID)).Scan(
		&folder.ID, &folder.Name, &folder.ParentFolderID, &folder.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmark folder by name: %w", err)
	}

	return folder, nil
}
