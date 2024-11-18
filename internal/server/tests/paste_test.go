package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/watzon/0x45/internal/server/services"
	"github.com/watzon/0x45/internal/server/tests/testutils"
)

func TestMultipartPasteUpload(t *testing.T) {
	env := testutils.SetupTestEnv(t)
	defer env.CleanupFn()

	uploadTestData := []struct {
		name           string
		content        string
		private        bool
		mimeType       string
		expectedStatus int
		withAuth       bool
		invalidAuth    bool
	}{
		{
			name:           "Valid text upload with auth",
			content:        "test content",
			private:        false,
			mimeType:       "text/plain; charset=utf-8", // ;charset=utf-8 is added by gabriel-vasile/mimetype if we don't explicitly set it
			expectedStatus: 200,
			withAuth:       true,
			invalidAuth:    false,
		},
		{
			name:           "Valid text upload without auth",
			content:        "test content",
			private:        false,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 200,
			withAuth:       false,
			invalidAuth:    false,
		},
		{
			name:           "Empty content",
			content:        "",
			private:        false,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 400,
			withAuth:       true,
			invalidAuth:    false,
		},
		{
			name:           "Private paste with auth",
			content:        "private content",
			private:        true,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 200,
			withAuth:       true,
			invalidAuth:    false,
		},
		{
			name:           "Private paste without auth",
			content:        "private content",
			private:        true,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 401,
			withAuth:       false,
			invalidAuth:    false,
		},
		{
			name:           "Private paste with invalid auth",
			content:        "private content",
			private:        true,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 401,
			withAuth:       true,
			invalidAuth:    true,
		},
		{
			name:           "Large content",
			content:        strings.Repeat("a", 1024*1024*9), // 9MB
			private:        false,
			mimeType:       "text/plain; charset=utf-8",
			expectedStatus: 200,
			withAuth:       false,
			invalidAuth:    false,
		},
		{
			name:           "Binary content",
			content:        string([]byte{0x00, 0x01, 0x02, 0x03}),
			private:        false,
			mimeType:       "application/octet-stream",
			expectedStatus: 200,
			withAuth:       false,
			invalidAuth:    false,
		},
		{
			name:           "Image content",
			content:        string([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}), // PNG header
			private:        false,
			mimeType:       "image/png",
			expectedStatus: 200,
			withAuth:       false,
			invalidAuth:    false,
		},
	}
	for _, tt := range uploadTestData {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			err := writer.WriteField("private", strconv.FormatBool(tt.private))
			require.NoError(t, err)

			err = writer.WriteField("content", tt.content)
			require.NoError(t, err)
			writer.Close()

			// Create request
			req := httptest.NewRequest("POST", "/p/", body)
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

			// If we got an unexpected status code, let's log the response body
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("Response body: %s", string(body))
				resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset the body for later use
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				var paste services.PasteResponse
				err = json.NewDecoder(resp.Body).Decode(&paste)
				require.NoError(t, err)
				assert.NotEmpty(t, paste.ID)
				assert.NotEmpty(t, paste.URL)
				assert.NotEmpty(t, paste.RawURL)
				assert.NotEmpty(t, paste.DownloadURL)
				assert.NotEmpty(t, paste.DeleteURL)
				assert.Equal(t, tt.private, paste.Private)
				assert.Equal(t, tt.mimeType, paste.MimeType)
			}
		})
	}
}

func TestJSONPasteUpload(t *testing.T) {
	env := testutils.SetupTestEnv(t)
	defer env.CleanupFn()

	uploadTestData := []struct {
		name           string
		body           string
		withAuth       bool
		invalidAuth    bool
		expectedStatus int
	}{
		{
			name:           "with content and private false",
			body:           `{"content": "test content", "private": false}`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 200,
		},
		{
			name:           "with empty content and private false",
			body:           `{"content": "", "private": false}`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
		{
			name:           "with content and private true",
			body:           `{"content": "test content", "private": true}`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 200,
		},
		{
			name:           "with invalid auth",
			body:           `{"content": "test content", "private": true}`,
			withAuth:       true,
			invalidAuth:    true,
			expectedStatus: 401,
		},
		{
			name:           "empty content",
			body:           `{"content": ""}`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
		{
			name:           "invalid json",
			body:           `{content: "test content")`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
		{
			name:           "empty json",
			body:           `{}`,
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
		{
			name:           "binary content",
			body:           fmt.Sprintf(`{"content": "%s"}`, string([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})),
			withAuth:       true,
			invalidAuth:    false,
			expectedStatus: 400,
		},
	}

	for _, tt := range uploadTestData {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("POST", "/p/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

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

			// If we got an unexpected status code, let's log the response body
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("Response body: %s", string(body))
				resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset the body for later use
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				// Parse body
				var body services.PasteOptions
				err = json.Unmarshal([]byte(tt.body), &body)
				require.NoError(t, err)

				// Parse response
				var paste services.PasteResponse
				err = json.NewDecoder(resp.Body).Decode(&paste)
				require.NoError(t, err)

				assert.NotEmpty(t, paste.ID)
				assert.NotEmpty(t, paste.URL)
				assert.NotEmpty(t, paste.RawURL)
				assert.NotEmpty(t, paste.DownloadURL)
				assert.NotEmpty(t, paste.DeleteURL)
				assert.Equal(t, body.Private, paste.Private)
			}
		})
	}
}

func TestPasteWithExpiresIn(t *testing.T) {
	env := testutils.SetupTestEnv(t)
	defer env.CleanupFn()

	uploadTestData := []struct {
		name           string
		content        string
		expiresIn      string
		expectedDelta  time.Duration
		private        bool
		expectedStatus int
	}{
		{
			name:           "with valid expires in",
			content:        "test content",
			expiresIn:      "1h",
			expectedDelta:  time.Hour,
			private:        false,
			expectedStatus: 200,
		},
		{
			name:           "with valid expires in (3 days)",
			content:        "test content",
			expiresIn:      "72h",
			expectedDelta:  72 * time.Hour,
			private:        false,
			expectedStatus: 200,
		},
		{
			name:           "with invalid expires in",
			content:        "test content",
			expiresIn:      "invalid",
			private:        false,
			expectedStatus: 400,
		},
	}

	for _, tt := range uploadTestData {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			err := writer.WriteField("private", strconv.FormatBool(tt.private))
			require.NoError(t, err)

			err = writer.WriteField("expires_in", tt.expiresIn)
			require.NoError(t, err)

			err = writer.WriteField("content", tt.content)
			require.NoError(t, err)
			writer.Close()

			// Create request
			req := httptest.NewRequest("POST", "/p/", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Record the time before making the request
			beforeRequest := time.Now()

			// Perform request
			resp, err := env.App.Test(req)
			require.NoError(t, err)

			// If we got an unexpected status code, let's log the response body
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("Response body: %s", string(body))
				resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset the body for later use
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				var paste services.PasteResponse
				err = json.NewDecoder(resp.Body).Decode(&paste)
				require.NoError(t, err)

				require.NotNil(t, paste.ExpiresAt, "ExpiresAt should not be nil")

				// Check that the expiry time is within 1 second of what we expect
				expectedTime := beforeRequest.Add(tt.expectedDelta)
				actualTime := *paste.ExpiresAt // Dereference the pointer for comparison
				timeDiff := actualTime.Sub(expectedTime)
				assert.Less(t, timeDiff.Abs(), time.Second, "Expiry time should be within 1 second of expected time")
			}
		})
	}
}
