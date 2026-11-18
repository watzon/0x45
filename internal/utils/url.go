package utils

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// isBlockedIP reports whether the given IP address must not be reached by a
// server-side fetch. It covers loopback, private, link-local (including the
// cloud metadata range 169.254.0.0/16), unspecified and other non-global
// ranges to prevent Server-Side Request Forgery.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	// Carrier-grade NAT range 100.64.0.0/10.
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 {
		return true
	}
	return false
}

func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(network, address string, _ interface{}) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if isBlockedIP(ip) {
				return fmt.Errorf("blocked address: %s", host)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
}

// GetContentFromURL fetches the raw content of a given URL. Only http and https
// URLs are permitted and requests to private, loopback or link-local addresses
// are rejected to prevent Server-Side Request Forgery.
func GetContentFromURL(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := newSafeHTTPClient().Do(req)
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
