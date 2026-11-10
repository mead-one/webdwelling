package bookmarkfile

import (
	"fmt"
	"io"
	"mime/multipart"

	"golang.org/x/net/html"
)

type ImportFolder struct {
	Name           string
	ChildFolders   []*ImportFolder
	ChildBookmarks []*ImportBookmark
}

type ImportBookmark struct {
	Title, URL, Tags string
}

func Parse(fileHeader *multipart.FileHeader) (*ImportFolder, error) {
	bookmarks := ImportFolder{}
	bookmarks.Name = fmt.Sprintf("import-%s", fileHeader.Filename)

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open bookmarks file: %w", err)
	}
	defer file.Close()

	var contentType string = fileHeader.Header.Get("Content-Type")
	fmt.Printf("CONTENT TYPE: %s\n", contentType)

	reader := io.Reader(file)

	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	if doc != nil {
		fmt.Println("Success")
	}

	return &bookmarks, nil
}
