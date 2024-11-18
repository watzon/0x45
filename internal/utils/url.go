package utils

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetContentFromURL fetches the raw content of a given URL
func GetContentFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// GetFilenameFromURL extracts the filename from the URL path
func GetFilenameFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// Get the last segment of the path
	segments := strings.Split(u.Path, "/")
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] != "" {
			return segments[i]
		}
	}

	return ""
}
