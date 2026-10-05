package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
)

var (
	ErrNotFullyCopied = errors.New("didn't copy entire file from the archive")
	ErrInvalidEntry   = errors.New("archive entry has an invalid file name")
)

// ExtractArchive extracts the archive at archivePath (which ends in .temp) if it
// contains exactly one file and returns the path of the extracted file. If the
// archive contains multiple files, the archive itself is returned instead.
func ExtractArchive(archivePath string) (string, error) {
	ctx := context.Background()

	// Our path will have a .temp appended to it so remove it before looking up the
	// extractor for the format.
	extractor, err := extractorFor(ctx, strings.TrimSuffix(archivePath, ".temp"))
	if err != nil {
		return "", err
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	var newPath string
	multipleFiles := false
	err = extractor.Extract(ctx, archive, func(ctx context.Context, f archives.FileInfo) error {
		if f.IsDir() {
			return nil
		}

		// Extract only one file per archive. Otherwise, stop walking,
		// remove extracted items, and deliver the archive itself.
		if newPath != "" {
			multipleFiles = true
			return fs.SkipAll
		}

		name, err := entryFileName(f.NameInArchive)
		if err != nil {
			return err
		}

		newPath = filepath.Join(filepath.Dir(archivePath), name+".temp")
		return extractFile(f, newPath)
	})

	if err != nil || multipleFiles {
		if newPath != "" {
			os.Remove(newPath)
		}
		if err != nil {
			return "", err
		}
		return archivePath, nil
	}

	// Archive didn't contain any files. Send the archive itself.
	if newPath == "" {
		return archivePath, nil
	}

	// If we extracted exactly one file, send that file and remove the archive.
	archive.Close()
	if err := os.Remove(archivePath); err != nil {
		log.Println("remove error", err)
	}
	return newPath, nil
}

func extractFile(f archives.FileInfo, destination string) error {
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}

	copied, err := io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if copied != f.Size() {
		return ErrNotFullyCopied
	}

	return nil
}

// entryFileName reduces a path inside of an archive to its base file name so
// that malicious entries can't write outside the download directory.
func entryFileName(nameInArchive string) (string, error) {
	name := path.Base(strings.ReplaceAll(nameInArchive, "\\", "/"))
	if name == "." || name == ".." || name == "/" || strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidEntry, nameInArchive)
	}
	return name, nil
}

// extractorFor returns an extractor for the archive format matching the file
// name's extension. The file contents are deliberately not inspected because
// formats like epub are zip files internally and must not be extracted.
func extractorFor(ctx context.Context, fileName string) (archives.Extractor, error) {
	format, _, err := archives.Identify(ctx, fileName, nil)
	if err != nil {
		return nil, err
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, fmt.Errorf("format of %s can't be extracted (%T)", fileName, format)
	}

	return extractor, nil
}

// IsArchive returns true if the file at the given path is an archive that can
// be extracted. Returns false otherwise.
func IsArchive(path string) bool {
	_, err := extractorFor(context.Background(), strings.TrimSuffix(path, ".temp"))
	return err == nil
}
