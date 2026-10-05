package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// List of file extensions that I've encountered.
// Some of them aren't eBooks, but they were returned
// in previous search results.
// When an extension is a prefix of another (azw/azw3, htm/html), the shorter
// one must come first so the longer, more specific match wins.
var fileTypes = [...]string{
	"epub",
	"mobi",
	"azw",
	"azw3",
	"htm",
	"html",
	"rtf",
	"pdf",
	"cdr",
	"lit",
	"cbr",
	"doc",
	"jpg",
	"txt",
	"rar", // Compressed extensions should always be last 2 items
	"zip",
}

// BookDetail contains the details of a single Book found on the IRC server
type BookDetail struct {
	Server string `json:"server"`
	Author string `json:"author"`
	Title  string `json:"title"`
	Format string `json:"format"`
	Size   string `json:"size"`
	Full   string `json:"full"`
}

type ParseError struct {
	Line  string `json:"line"`
	Error error  `json:"error"`
}

func (p *ParseError) MarshalJSON() ([]byte, error) {
	item := struct {
		Line  string `json:"line"`
		Error string `json:"error"`
	}{
		Line:  p.Line,
		Error: p.Error.Error(),
	}
	return json.Marshal(item)
}

func (p ParseError) String() string {
	return fmt.Sprintf("Error: %s. Line: %s.", p.Error, p.Line)
}

// ParseSearchFile converts a single search file into an array of BookDetail
func ParseSearchFile(filePath string) ([]BookDetail, []ParseError, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	books, errs := ParseSearchV2(file)
	return books, errs, nil
}

func ParseSearchV2(reader io.Reader) ([]BookDetail, []ParseError) {
	books := make([]BookDetail, 0)
	parseErrors := make([]ParseError, 0)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "!") {
			dat, err := parseLineV2(line)
			if err != nil {
				parseErrors = append(parseErrors, ParseError{Line: line, Error: err})
			} else {
				books = append(books, dat)
			}
		}
	}

	sort.Slice(books, func(i, j int) bool { return books[i].Server < books[j].Server })

	return books, parseErrors
}

func parseLineV2(line string) (BookDetail, error) {
	if !strings.HasPrefix(line, "!") {
		return BookDetail{}, errors.New("result lines must start with '!'")
	}

	firstSpace := strings.Index(line, " ")
	if firstSpace == -1 {
		return BookDetail{}, errors.New("unable parse server name")
	}
	server := line[1:firstSpace]

	size, endIndex := getSize(line)
	if endIndex <= firstSpace {
		return BookDetail{}, errors.New("unable to parse title")
	}
	body := strings.TrimSpace(line[firstSpace+1 : endIndex])

	// Handles case with weird author characters %\w% ("%F77FE9FF1CCD% Michael Haag")
	body = stripIdentifierPrefix(body)

	// Lines without an author look like "!server Title.epub"
	author := ""
	titleStart := 0
	if dashIndex := strings.Index(body, " - "); dashIndex != -1 {
		author = body[:dashIndex]
		titleStart = dashIndex + len(" - ")
	}

	title, format, ok := getTitle(body, titleStart)
	if !ok {
		return BookDetail{}, errors.New("unable to parse title")
	}

	return BookDetail{
		Server: server,
		Author: author,
		Title:  title,
		Format: format,
		Size:   size,
		Full:   strings.TrimSpace(line[:endIndex]),
	}, nil
}

// getSize returns the file size from the ::INFO:: block and the index where the
// block starts, or "N/A" and the line length if there isn't one.
func getSize(line string) (string, int) {
	const delimiter = " ::INFO:: "
	infoIndex := strings.LastIndex(line, delimiter)

	if infoIndex != -1 {
		// Handle cases when there is additional info after the file size (ex ::HASH:: )
		parts := strings.Split(line[infoIndex+len(delimiter):], " ")
		return parts[0], infoIndex
	}

	return "N/A", len(line)
}

// stripIdentifierPrefix removes a leading "%HEX% " identifier some servers add.
func stripIdentifierPrefix(body string) string {
	if !strings.HasPrefix(body, "%") {
		return body
	}
	closing := strings.Index(body[1:], "% ")
	if closing == -1 {
		return body
	}
	return strings.TrimSpace(body[closing+len("% ")+1:])
}

// getTitle finds the title (text between titleStart and the file extension) and
// the book's format.
func getTitle(body string, titleStart int) (string, string, bool) {
	title := ""
	fileFormat := ""
	found := false

	for _, ext := range fileTypes { //Loop through each possible file extension we've got on record
		endTitle := strings.Index(body[titleStart:], "."+ext) // check if it contains our extension
		if endTitle == -1 {
			continue
		}
		endTitle += titleStart

		fileFormat = ext
		if ext == "rar" || ext == "zip" { // If the extension is .rar or .zip the actual format is contained in ()
			for _, ext2 := range fileTypes[:len(fileTypes)-2] { // Range over the eBook formats (exclude archives)
				if strings.Contains(strings.ToLower(body[:endTitle]), ext2) {
					fileFormat = ext2
				}
			}
		}
		// "Title.epub.rar" -> "Title"
		title = strings.TrimSuffix(body[titleStart:endTitle], "."+fileFormat)
		found = true
	}

	return title, fileFormat, found
}
