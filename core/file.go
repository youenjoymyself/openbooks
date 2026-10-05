package core

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/evan-buss/openbooks/dcc"
	"github.com/evan-buss/openbooks/util"
)

var ErrOutsideDirectory = errors.New("refusing to write file outside of the download directory")

func DownloadExtractDCCString(baseDir, dccStr string, progress io.Writer) (string, error) {
	// Download the file and wait until it is completed
	download, err := dcc.ParseString(dccStr)
	if err != nil {
		return "", err
	}

	dccPath := filepath.Join(baseDir, download.Filename+".temp")
	if filepath.Dir(dccPath) != filepath.Clean(baseDir) {
		return "", ErrOutsideDirectory
	}

	file, err := os.Create(dccPath)
	if err != nil {
		return "", err
	}

	writer := io.Writer(file)
	if progress != nil {
		writer = io.MultiWriter(file, progress)
	}

	// Download DCC data to the file
	err = download.Download(writer)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(dccPath)
		return "", err
	}

	if !util.IsArchive(dccPath) {
		return renameTempFile(dccPath)
	}

	extractedPath, err := util.ExtractArchive(dccPath)
	if err != nil {
		os.Remove(dccPath)
		return "", err
	}

	return renameTempFile(extractedPath)
}

func renameTempFile(filePath string) (string, error) {
	if filepath.Ext(filePath) != ".temp" {
		return filePath, nil
	}

	newPath := filePath[:len(filePath)-len(".temp")]
	if err := os.Rename(filePath, newPath); err != nil {
		os.Remove(filePath)
		return "", err
	}
	return newPath, nil
}
