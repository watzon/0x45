package utils

import (
	"io"
	"net/http"
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
