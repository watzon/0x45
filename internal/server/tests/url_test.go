package tests

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/watzon/0x45/internal/server/services"
	"github.com/watzon/0x45/internal/server/tests/testutils"
)

func TestMultipartCreateShortlink(t *testing.T) {
	env := testutils.SetupTestEnv(t)
	defer env.CleanupFn()

	testData := []struct {
		name           string
		url            string
		title          string
		withAuth       bool
		invalidAuth    bool
		expectedStatus int
	}{
		{
			name:           "with valid url",
			url:            "https://google.com",
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 200,
		},
		{
			name:           "with invalid url",
			url:            "invalid url",
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			err := writer.WriteField("url", tt.url)
			require.NoError(t, err)

			if tt.title != "" {
				err = writer.WriteField("title", tt.title)
				require.NoError(t, err)
			}

			err = writer.Close()
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest("POST", "/u/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			if tt.withAuth {
				if tt.invalidAuth {
					req.Header.Set("Authorization", "Bearer invalid-api-key")
				} else {
					req.Header.Set("Authorization", "Bearer test-api-key")
				}
			}

			// Perform request
			resp, err := env.App.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				// Check response
				var response services.ShortlinkResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.ID)
				assert.NotEmpty(t, response.URL)
				assert.NotEmpty(t, response.Title)
				assert.NotEmpty(t, response.ShortURL)
				assert.NotEmpty(t, response.StatsURL)
				assert.NotEmpty(t, response.DeleteURL)
			}
		})
	}
}
