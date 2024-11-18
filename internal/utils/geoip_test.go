package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockTransport struct {
	response string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(t.response)),
	}, nil
}

func TestGetLocationInfo(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		mock     string
		expected LocationInfo
	}{
		{
			name: "valid ip address",
			ip:   "136.36.156.245",
			mock: `{"status":"success","country":"United States","countryCode":"US","region":"UT","regionName":"Utah","city":"Salt Lake City","zip":"84106","lat":40.6982,"lon":-111.841,"timezone":"America/Denver","isp":"Google Fiber Inc.","org":"Google Fiber Inc","as":"AS16591 Google Fiber Inc.","query":"136.36.156.245"}`,
			expected: LocationInfo{
				City:    "Salt Lake City",
				Region:  "Utah",
				ZipCode: "84106",
				Country: "US",
			},
		},
		{
			name: "invalid ip address",
			ip:   "invalid",
			mock: `{"status":"fail","message":"invalid query","query":"invalid"}`,
			expected: LocationInfo{
				City:    "",
				Region:  "",
				ZipCode: "",
				Country: "",
			},
		},
		{
			name: "server error",
			ip:   "error",
			mock: `{"error": "internal server error"}`,
			expected: LocationInfo{
				City:    "",
				Region:  "",
				ZipCode: "",
				Country: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a custom client with our mock transport
			client := &http.Client{
				Transport: &mockTransport{response: tt.mock},
			}

			// Create a test server just to get a valid URL
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			defer server.Close()

			// Call the function being tested
			result := GetLocationInfoWithClient(tt.ip, client)

			// Check the results
			if result.City != tt.expected.City {
				t.Errorf("City = %v, want %v", result.City, tt.expected.City)
			}
			if result.Region != tt.expected.Region {
				t.Errorf("Region = %v, want %v", result.Region, tt.expected.Region)
			}
			if result.ZipCode != tt.expected.ZipCode {
				t.Errorf("ZipCode = %v, want %v", result.ZipCode, tt.expected.ZipCode)
			}
			if result.Country != tt.expected.Country {
				t.Errorf("Country = %v, want %v", result.Country, tt.expected.Country)
			}
		})
	}
}
