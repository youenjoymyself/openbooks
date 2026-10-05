package dcc

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/evan-buss/openbooks/mock"
)

// TestStringParsing makes sure that data is properly extracted from the DCC
// response string. (filename, IP conversion, port, and size)
func TestStringParsing(t *testing.T) {
	tables := []struct {
		search   string
		download *Download
	}{
		{
			":SearchOok!ook@only.ook PRIVMSG evan_28 :DCC SEND SearchOok_results_for__hp_lovecraft.txt.zip 1543751478 2043 784",
			&Download{Filename: "SearchOok_results_for__hp_lovecraft.txt.zip", IP: "92.3.199.54", Port: "2043", Size: 784},
		},
		{
			":Search!Search@ihw-4q5hcb.dyn.suddenlink.net PRIVMSG evan_bot :DCC SEND SearchBot_results_for__stephen_king_the_stand.txt.zip 2907707975 4342 1116",
			&Download{Filename: "SearchBot_results_for__stephen_king_the_stand.txt.zip", IP: "173.80.26.71", Port: "4342", Size: 1116},
		},
		{
			`:DV8!HandyAndy@ihw-39fkft.ip-164-132-173.eu PRIVMSG negative-bishop-1 :DCC SEND "Douglas Adams - [HITCHHIKER'S GUIDE TO THE GALAXY & THE 01] - Hitchhiker's Guide to the Galaxy & The (v5.0) (EPUB).rar" 2760158537 2050 2321788`,
			&Download{Filename: "Douglas Adams - [HITCHHIKER'S GUIDE TO THE GALAXY & THE 01] - Hitchhiker's Guide to the Galaxy & The (v5.0) (EPUB).rar", IP: "164.132.173.73", Port: "2050", Size: 2321788},
		},
	}

	for _, table := range tables {
		download, err := ParseString(table.search)
		require.NoError(t, err)
		assert.Equal(t, table.download, download)
	}
}

func TestDownload(t *testing.T) {
	text := "Test dcc download content."

	textDownload := Download{
		Filename: "test.txt",
		IP:       "localhost",
		Port:     "6969",
		Size:     int64(len(text)),
	}

	reader := bytes.NewReader([]byte(text))
	server := mock.DccServer{
		Port:   ":" + textDownload.Port,
		Reader: reader,
	}

	ready := make(chan struct{}, 1)
	go server.Start(ready)
	<-ready

	t.Log("After server start")

	received := new(mock.WriteCloser)
	err := textDownload.Download(received)
	require.NoError(t, err)
	assert.Equal(t, text, string(received.Data))
}

func TestFilenameSanitization(t *testing.T) {
	cases := map[string]string{
		`DCC SEND ../../.bashrc 2130706433 1 1`:                    ".bashrc",
		`DCC SEND /etc/passwd 2130706433 1 1`:                      "passwd",
		`DCC SEND "..\..\Windows\evil.dll" 2130706433 1 1`:         "evil.dll",
		`DCC SEND "dir/My Book - Author.epub" 2130706433 1 1`:      "My Book - Author.epub",
		`DCC SEND "Douglas Adams - Hitchhiker.rar" 2130706433 1 1`: "Douglas Adams - Hitchhiker.rar",
	}
	for input, expected := range cases {
		download, err := ParseString(input)
		require.NoError(t, err, input)
		assert.Equal(t, expected, download.Filename, input)
	}

	for _, input := range []string{
		`DCC SEND .. 2130706433 1 1`,
		`DCC SEND "../" 2130706433 1 1`,
	} {
		_, err := ParseString(input)
		assert.ErrorIs(t, err, ErrInvalidFilename, input)
	}

	_, err := ParseString(`DCC SEND / 2130706433 1 1`)
	assert.Error(t, err)
}

// A size of 0 means unknown. Everything until the sender closes the
// connection should be received.
func TestDownloadUnknownSize(t *testing.T) {
	text := "Unknown size content."
	server := mock.DccServer{
		Port:   "127.0.0.1:6970",
		Reader: bytes.NewReader([]byte(text)),
	}
	ready := make(chan struct{}, 1)
	go server.Start(ready)
	<-ready

	download := Download{Filename: "test.txt", IP: "127.0.0.1", Port: "6970", Size: 0}
	received := new(mock.WriteCloser)
	require.NoError(t, download.Download(received))
	assert.Equal(t, text, string(received.Data))
}

// Senders that send more than the advertised size must not grow the file.
func TestDownloadTruncatesExtraData(t *testing.T) {
	text := "0123456789EXTRA"
	server := mock.DccServer{
		Port:   "127.0.0.1:6972",
		Reader: bytes.NewReader([]byte(text)),
	}
	ready := make(chan struct{}, 1)
	go server.Start(ready)
	<-ready

	download := Download{Filename: "test.txt", IP: "127.0.0.1", Port: "6972", Size: 10}
	received := new(mock.WriteCloser)
	require.NoError(t, download.Download(received))
	assert.Equal(t, "0123456789", string(received.Data))
}
