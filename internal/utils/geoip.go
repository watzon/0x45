package utils

import (
	"encoding/json"
	"net/http"
)

type LocationInfo struct {
	City    string `json:"city"`
	Region  string `json:"regionName"`
	ZipCode string `json:"zip"`
	Country string `json:"countryCode"`
}

// GetLocationInfo fetches the location info for an IP address using ip-api.com
func GetLocationInfo(ipAddress string) LocationInfo {
	return GetLocationInfoWithClient(ipAddress, http.DefaultClient)
}

// GetLocationInfoWithClient fetches the location info using a custom HTTP client
func GetLocationInfoWithClient(ipAddress string, client *http.Client) LocationInfo {
	resp, err := client.Get("http://ip-api.com/json/" + ipAddress)
	if err != nil {
		return LocationInfo{}
	}
	defer resp.Body.Close()

	var result struct {
		City    string `json:"city"`
		Region  string `json:"regionName"`
		ZipCode string `json:"zip"`
		Country string `json:"countryCode"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return LocationInfo{}
	}

	return result
}
