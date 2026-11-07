package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

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
func InitDB(dataDir string) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
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

	err = migrate()
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
}

// Close the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// Run migrations not present in live database
func migrate() error {
	files, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return err
	}

	var targetVersion int = len(files)
	var currentVersion int
	err = DB.QueryRow("PRAGMA user_version;").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to read user_version from database: %w", err)
	}

	// Check that files are correctly named by schema version
	for i, file := range files {
		version := i + 1

		pattern := fmt.Sprintf("%04d_*.sql", version)
		matched, err := filepath.Match(pattern, file.Name())
		if err != nil {
			return fmt.Errorf("failed to check migration filename %s: %w", file.Name(), err)
		}

		if !matched {
			return fmt.Errorf("filename %s isn't correct for schema version %d", file.Name(), version)
		}
	}

	if currentVersion > targetVersion {
		return fmt.Errorf("current schema version (%d) is newer than this build's latest migration (%d)", currentVersion, targetVersion)
	}
	if currentVersion == targetVersion {
		return nil
	}

	for i, file := range files[currentVersion:] {
		version := currentVersion + i + 1

		err := applyMigration(version, file.Name())
		if err != nil {
			return err
		}
		log.Printf("Successfully applied migration: %q", file.Name())
	}

	return nil
}

func applyMigration(version int, name string) error {
	sqlContent, err := fs.ReadFile(migrationFiles, path.Join("migrations", name))
	if err != nil {
		return fmt.Errorf("failed to read migration file %d from go:embed: %w", version, err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to create database transaction: %w", err)
	}
	defer tx.Rollback() // Rollback has no effect if Commit successfully ran

	_, err = tx.Exec(string(sqlContent))
	if err != nil {
		return fmt.Errorf("failed to apply migration version %d: %w", version, err)
	}
	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version))
	if err != nil {
		return fmt.Errorf("failed to update user_version to %d: %w", version, err)
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit database transaction for migration %d: %w", version, err)
	}

	return nil
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
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	ParentFolderID *int64            `json:"parent_folder_id,omitempty"`
	Public         bool              `json:"public"`
	CreatedAt      string            `json:"created_at"`
	ChildFolders   []*BookmarkFolder `json:"child_folders"`
	ChildBookmarks []*Bookmark       `json:"child_bookmarks"`
}

type Bookmark struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Tags      string `json:"tags"`
	FolderID  *int64 `json:"folder_id,omitempty"`
	Public    bool   `json:"public"`
	CreatedAt string `json:"created_at"`
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

func RenameBookmarkFolder(userID int, folderID int64, isPublic bool, name string) (*BookmarkFolder, error) {
	folder := &BookmarkFolder{}

	err := DB.QueryRow(`
	UPDATE bookmark_folders
	SET name = :name, public = :public
	WHERE id = :folder_id AND user_id = :user_id
	RETURNING id,name,parent_folder_id,public,created_at
	`, sql.Named("name", name), sql.Named("public", isPublic), sql.Named("folder_id", folderID), sql.Named("user_id", userID),
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
