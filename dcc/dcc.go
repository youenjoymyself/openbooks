package dcc

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// There are two types of DCC strings this program accepts.
// Download contains all of the necessary DCC info parsed from the DCC SEND string

var (
	ErrInvalidDCCString = errors.New("invalid dcc send string")
	ErrInvalidIP        = errors.New("unable to convert int IP to string")
	ErrMissingBytes     = errors.New("download size didn't match dcc file size. data could be missing")
	ErrInvalidFilename  = errors.New("invalid dcc file name")
)

const (
	// Maximum time to wait for the sender to accept the connection.
	dialTimeout = 30 * time.Second
	// Maximum time to wait between received chunks before giving up.
	idleTimeout = 2 * time.Minute
)

var dccRegex = regexp.MustCompile(`DCC SEND "?(.+[^"])"?\s(\d+)\s+(\d+)\s+(\d+)\s*`)

type Download struct {
	Filename string
	IP       string
	Port     string
	Size     int64
}

// ParseString parses the important data of a DCC SEND string
func ParseString(text string) (*Download, error) {
	groups := dccRegex.FindStringSubmatch(text)

	if len(groups) == 0 {
		return nil, ErrInvalidDCCString
	}

	ip, err := stringToIP(groups[2])
	if err != nil {
		return nil, err
	}

	size, err := strconv.ParseInt(groups[4], 10, 64)
	if err != nil {
		return nil, err
	}

	filename, err := sanitizeFilename(groups[1])
	if err != nil {
		return nil, err
	}

	return &Download{
		Filename: filename,
		IP:       ip,
		Port:     groups[3],
		Size:     size,
	}, nil
}

// Download writes the data contained in the DCC Download
func (download Download) Download(writer io.Writer) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(download.IP, download.Port), dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	// NOTE: Not using the idiomatic io.Copy or io.CopyBuffer because they are
	// much slower in real world tests than the manual way. I suspect it has to
	// do with the way the DCC server is sending data. I don't think it ever sends
	// an EOF like the io.* methods expect.

	// Benchmark: 2.36MB File
	// CopyBuffer - 4096 - 2m32s, 2m18s, 2m32s
	// Copy - 2m35s
	// Custom - 1024 - 35s
	// Custom - 4096 - 46s, 14s
	var received int64
	bytes := make([]byte, 4096)
	// A size of 0 means the sender didn't specify one. Read until EOF.
	for download.Size == 0 || received < download.Size {
		conn.SetReadDeadline(time.Now().Add(idleTimeout))
		n, err := conn.Read(bytes)

		// Never write more than the advertised size.
		if download.Size > 0 && received+int64(n) > download.Size {
			n = int(download.Size - received)
		}

		if n > 0 {
			if _, writeErr := writer.Write(bytes[:n]); writeErr != nil {
				return writeErr
			}
			received += int64(n)
		}

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}

	if download.Size == 0 {
		return nil
	}

	if received != download.Size {
		return ErrMissingBytes
	}

	return nil
}

// sanitizeFilename reduces a sender-supplied file name to a single path
// element so that it can't be used to write outside the download directory.
func sanitizeFilename(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "." || name == ".." || name == "/" || strings.TrimSpace(name) == "" {
		return "", ErrInvalidFilename
	}
	return name, nil
}

// Convert a given 32 bit IP integer to an IP string
// Ex) 2907707975 -> 192.168.1.1
func stringToIP(nn string) (string, error) {
	temp, err := strconv.ParseUint(nn, 10, 32)
	if err != nil {
		return "", ErrInvalidIP
	}
	intIP := uint32(temp)

	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, intIP)
	return ip.String(), nil
}
