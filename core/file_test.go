package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/evan-buss/openbooks/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A malicious DCC file name must not write outside of the download directory.
func TestDownloadStaysInDirectory(t *testing.T) {
	content := "malicious content"
	server := mock.DccServer{Port: "127.0.0.1:6971", Reader: bytes.NewReader([]byte(content))}
	ready := make(chan struct{}, 1)
	go server.Start(ready)
	<-ready

	root := t.TempDir()
	downloadDir := filepath.Join(root, "books")
	require.NoError(t, os.Mkdir(downloadDir, 0755))

	dccString := `DCC SEND "../escaped.txt" 2130706433 6971 17`
	path, err := DownloadExtractDCCString(downloadDir, dccString, nil)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(downloadDir, "escaped.txt"), path)
	assert.NoFileExists(t, filepath.Join(root, "escaped.txt"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

// Failed downloads must not leave .temp files behind.
func TestFailedDownloadCleansUp(t *testing.T) {
	downloadDir := t.TempDir()

	// Nothing is listening on this port.
	_, err := DownloadExtractDCCString(downloadDir, "DCC SEND book.epub 2130706433 1 100", nil)
	assert.Error(t, err)

	entries, err := os.ReadDir(downloadDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
