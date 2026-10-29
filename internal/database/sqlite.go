package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDuplicateName = errors.New("duplicate name")
	ErrInvalidMove   = errors.New("invalid move")
)

// Return a UNIQUE constraint failure as ErrDuplicateName, and other errors as is
func uniqueAsDuplicate(err error) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		return ErrDuplicateName
	}
	return err
}

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
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        tags TEXT NOT NULL DEFAULT '',
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
	ID             int64
	Name           string
	ParentFolderID *int64
	Public         bool
	CreatedAt      string
	ChildFolders   []*BookmarkFolder
	ChildBookmarks []*Bookmark
}

type Bookmark struct {
	ID        int64
	Title     string
	URL       string
	Tags      string
	FolderID  *int64
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
	folderMap := make(map[int64]*BookmarkFolder)
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
		var parentFolderID *int64

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
		var folderID *int64

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
func AddBookmark(userID int, title string, url string, tags string, folderID *int64, public bool) (*Bookmark, error) {
	// Write bookmark to database
	result, err := DB.Exec(`
		INSERT INTO bookmarks (user_id,title,url,tags,folder_id,public)
		SELECT :user_id, :title, :url, :tags, :folder_id, :public
		WHERE :folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders WHERE id = :folder_id AND user_id = :user_id)
		`, sql.Named("user_id", userID), sql.Named("title", title), sql.Named("url", url),
		sql.Named("tags", tags), sql.Named("folder_id", folderID), sql.Named("public", public),
	)
	if err != nil {
		return nil, fmt.Errorf("create bookmark: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("create bookmark: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("create bookmark: %w", ErrNotFound)
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create bookmark: %w", err)
	}

	bookmark := &Bookmark{
		ID:        newID,
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
func MoveBookmark(userID int, bookmarkID int64, folderID *int64) error {
	result, err := DB.Exec(`
	UPDATE bookmarks SET folder_id = :folder_id
	WHERE id = :bookmark_id AND user_id = :user_id
		AND (:folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders WHERE id = :folder_id AND user_id = :user_id))
	`, sql.Named("folder_id", folderID), sql.Named("bookmark_id", bookmarkID), sql.Named("user_id", userID))
	if err != nil {
		return fmt.Errorf("move bookmark %d: %w", bookmarkID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("move bookmark %d: %w", bookmarkID, err)
	}
	if n == 0 {
		return fmt.Errorf("move bookmark %d: %w", bookmarkID, ErrNotFound)
	}

	return nil
}

// Edit a bookmark
func EditBookmark(userID int, bookmarkID int64, title string, url string, tags string, folderID *int64, public bool) (*Bookmark, error) {
	bookmark := &Bookmark{}
	err := DB.QueryRow(`
		UPDATE bookmarks SET title = :title, url = :url, tags = :tags, folder_id = :folder_id, public = :public
		WHERE id = :bookmark_id AND user_id = :user_id
		AND (:folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders
				WHERE id = :folder_id AND user_id = :user_id))
		RETURNING id,title,url,tags,folder_id,public,created_at
	`, sql.Named("user_id", userID), sql.Named("bookmark_id", bookmarkID),
		sql.Named("title", title), sql.Named("url", url),
		sql.Named("tags", tags), sql.Named("folder_id", folderID),
		sql.Named("public", public)).Scan(&bookmark.ID, &bookmark.Title, &bookmark.URL,
		&bookmark.Tags, &bookmark.FolderID, &bookmark.Public, &bookmark.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("edit bookmark %d: %w", bookmarkID, ErrNotFound)
	} else if err != nil {
		return nil, fmt.Errorf("edit bookmark %d: %w", bookmarkID, err)
	}

	return bookmark, nil
}

// Delete a bookmark
func DeleteBookmark(userID int, bookmarkID int64) error {
	result, err := DB.Exec(`
	DELETE FROM bookmarks
	WHERE id = :bookmark_id AND user_id = :user_id
	`, sql.Named("bookmark_id", bookmarkID), sql.Named("user_id", userID))
	// Create bookmark struct
	if err != nil {
		return fmt.Errorf("delete bookmark %d: %w", bookmarkID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bookmark %d: %w", bookmarkID, err)
	}
	if n == 0 {
		return fmt.Errorf("delete bookmark %d: %w", bookmarkID, ErrNotFound)
	}

	return nil
}

// Add a new bookmark folder
func AddBookmarkFolder(userID int, name string, parentFolderID *int64, public bool) (*BookmarkFolder, error) {
	result, err := DB.Exec(`
		INSERT INTO bookmark_folders (user_id,name,parent_folder_id,public)
		SELECT :user_id, :name, :parent_folder_id, :public
		WHERE :parent_folder_id IS NULL
			OR EXISTS (SELECT 1 FROM bookmark_folders
				WHERE user_id = :user_id AND id = :parent_folder_id)
		`, sql.Named("user_id", userID), sql.Named("name", name),
		sql.Named("parent_folder_id", parentFolderID), sql.Named("public", public),
	)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", uniqueAsDuplicate(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("create folder: %w", ErrNotFound)
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	folder := &BookmarkFolder{
		ID:             newID,
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
func MoveBookmarkFolder(userID int, folderID int64, destinationFolderID *int64) error {
	// Check if the destination folder is a descendant of the moved folder
	if FolderIsDescendant(folderID, destinationFolderID) {
		return fmt.Errorf("move folder %d: %w", folderID, ErrInvalidMove)
	}

	result, err := DB.Exec(`
	UPDATE bookmark_folders
	SET parent_folder_id = :destination_folder_id
	WHERE id = :folder_id AND user_id = :user_id
		AND (:destination_folder_id IS NULL
			OR EXISTS ( SELECT 1 FROM bookmark_folders
				WHERE id = :destination_folder_id AND user_id = :user_id))
		`, sql.Named("destination_folder_id", destinationFolderID), sql.Named("folder_id", folderID), sql.Named("user_id", userID))
	if err != nil {
		return fmt.Errorf("move folder %d: %w", folderID, uniqueAsDuplicate(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("move folder %d: %w", folderID, err)
	}
	if n == 0 {
		return fmt.Errorf("move folder %d: %w", folderID, ErrNotFound)
	}

	return nil
}

// Check if a folder is a descendant of another folder
func FolderIsDescendant(folderID int64, destinationFolderID *int64) bool {
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

func RenameBookmarkFolder(userID int, folderID int64, name string) (*BookmarkFolder, error) {
	folder := &BookmarkFolder{}

	err := DB.QueryRow(`
	UPDATE bookmark_folders
	SET name = :name
	WHERE id = :folder_id AND user_id = :user_id
	RETURNING id,name,parent_folder_id,public,created_at
	`, sql.Named("name", name), sql.Named("folder_id", folderID), sql.Named("user_id", userID),
	).Scan(&folder.ID, &folder.Name, &folder.ParentFolderID, &folder.Public, &folder.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("rename folder %d: %w", folderID, ErrNotFound)
	} else if err != nil {
		return nil, fmt.Errorf("rename folder %d: %w", folderID, uniqueAsDuplicate(err))
	}

	return folder, nil
}

func DeleteBookmarkFolder(userID int, folderID int64) error {
	// Delete the folder, if it belongs to the user
	result, err := DB.Exec(`
	DELETE FROM bookmark_folders WHERE id = :folder_id AND user_id = :user_id
	`, sql.Named("folder_id", folderID), sql.Named("user_id", userID))
	if err != nil {
		return fmt.Errorf("delete folder(s) %d: %w", folderID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete folder(s) %d: %w", folderID, err)
	}
	if n == 0 {
		return fmt.Errorf("delete folder(s) %d: %w", folderID, ErrNotFound)
	}
	// Database cascade handles all descendant folders and bookmarks

	return nil
}

// Get a bookmark folder by name
func GetBookmarkFolderByID(folderID int64) (*BookmarkFolder, error) {
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
