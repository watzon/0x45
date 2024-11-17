package services

import (
	"io"
	"mime/multipart"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
)

// RequestParser handles unified request parsing for different content types
type RequestParser struct {
	ctx *fiber.Ctx
}

// NewRequestParser creates a new RequestParser instance
func NewRequestParser(c *fiber.Ctx) *RequestParser {
	return &RequestParser{ctx: c}
}

// ParseUploadRequest parses various types of upload requests into a unified format
func (p *RequestParser) ParseUploadRequest() (*UploadRequest, error) {
	contentType := p.ctx.Get("Content-Type")

	// Handle multipart form uploads
	if form, err := p.ctx.MultipartForm(); err == nil {
		return p.parseMultipartUpload(form)
	}

	// Handle JSON uploads
	if contentType == "application/json" {
		return p.parseJSONUpload()
	}

	// Handle raw body uploads
	return p.parseRawUpload()
}

// ParseJSON attempts to parse the request body as JSON into the provided struct
func (p *RequestParser) ParseJSON(out interface{}) error {
	if err := p.ctx.BodyParser(out); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON payload")
	}
	return nil
}

// Helper functions

func (p *RequestParser) parseMultipartUpload(form *multipart.Form) (*UploadRequest, error) {
	// Get file from form
	var file *multipart.FileHeader
	if len(form.File["file"]) > 0 {
		file = form.File["file"][0]
	} else {
		return nil, fiber.NewError(fiber.StatusBadRequest, "No file provided")
	}

	// Open and read file
	f, err := file.Open()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to open uploaded file")
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read file content")
	}

	// Check for empty content
	if len(content) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Empty file")
	}

	// Get other form values
	return &UploadRequest{
		Content:     content,
		Filename:    file.Filename,
		Extension:   p.ctx.FormValue("extension", ""),
		ExpiresIn:   p.ctx.FormValue("expires_in", ""),
		Private:     p.ctx.FormValue("private") == "true",
		ContentType: file.Header.Get("Content-Type"),
	}, nil
}

func (p *RequestParser) parseJSONUpload() (*UploadRequest, error) {
	var req struct {
		Content   string `json:"content"`
		URL       string `json:"url"`
		Filename  string `json:"filename"`
		Extension string `json:"extension"`
		ExpiresIn string `json:"expires_in"`
		Private   bool   `json:"private"`
	}

	if err := p.ParseJSON(&req); err != nil {
		return nil, err
	}

	// If URL is provided, create a URL-based upload
	if req.URL != "" {
		return &UploadRequest{
			URL:       req.URL,
			Filename:  req.Filename,
			Extension: req.Extension,
			ExpiresIn: req.ExpiresIn,
			Private:   req.Private,
		}, nil
	}

	// Otherwise, expect content in the request
	if req.Content == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Content or URL is required")
	}

	content := []byte(req.Content)
	mime := mimetype.Detect(content)

	return &UploadRequest{
		Content:     content,
		Filename:    req.Filename,
		Extension:   req.Extension,
		ExpiresIn:   req.ExpiresIn,
		Private:     req.Private,
		ContentType: mime.String(),
	}, nil
}

func (p *RequestParser) parseRawUpload() (*UploadRequest, error) {
	content := p.ctx.Body()
	if len(content) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Empty request body")
	}

	mime := mimetype.Detect(content)
	filename := p.ctx.Get("X-Filename", "")
	extension := p.ctx.Get("X-Extension", "")

	return &UploadRequest{
		Content:     content,
		Filename:    filename,
		Extension:   extension,
		ExpiresIn:   p.ctx.Get("X-Expires-In", ""),
		Private:     p.ctx.Get("X-Private") == "true",
		ContentType: mime.String(),
	}, nil
}
