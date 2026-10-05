package util

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createZip(t *testing.T, path string, files map[string]string) {
	out, err := os.Create(path)
	require.NoError(t, err)
	defer out.Close()

	writer := zip.NewWriter(out)
	for name, content := range files {
		w, err := writer.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
}

func TestExtractSingleFile(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "results.txt.zip.temp")
	createZip(t, archive, map[string]string{"results.txt": "search results"})

	path, err := ExtractArchive(archive)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "results.txt.temp"), path)
	assert.NoFileExists(t, archive, "archive is removed after extraction")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "search results", string(data))
}

func TestExtractMultipleFilesReturnsArchive(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "books.zip.temp")
	createZip(t, archive, map[string]string{"one.epub": "1", "two.epub": "2"})

	path, err := ExtractArchive(archive)
	require.NoError(t, err)

	assert.Equal(t, archive, path)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "partially extracted files are removed")
}

// Archive entries with paths must be extracted into the download directory.
func TestExtractPathTraversal(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "books")
	require.NoError(t, os.Mkdir(dir, 0755))
	archive := filepath.Join(dir, "evil.zip.temp")
	createZip(t, archive, map[string]string{"../../escaped.txt": "evil"})

	path, err := ExtractArchive(archive)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "escaped.txt.temp"), path)
	assert.NoFileExists(t, filepath.Join(root, "escaped.txt.temp"))
}

func TestIsArchive(t *testing.T) {
	assert.True(t, IsArchive("results.txt.zip.temp"))
	assert.True(t, IsArchive("book.rar"))
	// epub files are zip files internally but must be delivered as is.
	assert.False(t, IsArchive("book.epub.temp"))
	assert.False(t, IsArchive("book.mobi"))
	assert.False(t, IsArchive("book.pdf.temp"))
}
