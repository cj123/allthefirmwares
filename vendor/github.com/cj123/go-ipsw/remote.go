package ipsw

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"howett.net/ranger"
)

var MaxDownloadTries = 10

var DefaultClient HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// HTTPClient is a wrapper for client
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
	Get(string) (*http.Response, error)
	Head(string) (*http.Response, error)
	Post(string, string, io.Reader) (*http.Response, error)
	PostForm(string, url.Values) (*http.Response, error)
}

func bufferedDownload(file *zip.File, writer io.Writer) error {
	rc, err := file.Open()

	if err != nil {
		return err
	}

	defer rc.Close()

	_, err = io.Copy(writer, rc)

	return err
}

func DownloadFile(resource, file string, w io.Writer) error {
	u, err := url.Parse(resource)

	if err != nil {
		return err
	}

	var zipReader *zip.Reader

	for downloadCount := 1; downloadCount <= MaxDownloadTries; downloadCount++ {
		reader, err := ranger.NewReader(
			&ranger.HTTPRanger{
				URL:    u,
				Client: DefaultClient,
			},
		)

		if err != nil {
			return err
		}

		readerLen, err := reader.Length()

		if err != nil {
			return err
		}

		zipReader, err = zip.NewReader(reader, readerLen)

		if err == zip.ErrFormat && downloadCount != MaxDownloadTries {
			log.Printf("Caught error, %s, trying again (%d of %d)", err, downloadCount, MaxDownloadTries)
			continue
		} else if err != nil {
			return err
		} else { // err == nil
			break
		}
	}

	for _, f := range zipReader.File {
		if f.Name == file {
			return bufferedDownload(f, w)
		}
	}

	return fmt.Errorf("pwn: file '%s' not found in resource '%s'", file, resource)
}
