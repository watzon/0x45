package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/watzon/0x45/internal/config"
	"github.com/watzon/0x45/internal/server/services"
	"go.uber.org/zap"
)

type PasteHandlers struct {
	services *services.Services
	logger   *zap.Logger
	config   *config.Config
}

func NewPasteHandlers(services *services.Services, logger *zap.Logger, config *config.Config) *PasteHandlers {
	return &PasteHandlers{
		services: services,
		logger:   logger,
		config:   config,
	}
}

// @id HandleUpload
// @Summary Upload a new paste
// @Tags Paste
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {object} services.PasteResponse
// @Failure 400 {object} fiber.Error
func (h *PasteHandlers) HandleUpload(c *fiber.Ctx) error {
	return h.services.Paste.UploadPaste(c)
}

// HandleView serves the content with syntax highlighting if applicable
func (h *PasteHandlers) HandleView(c *fiber.Ctx) error {
	id := getPasteID(c)

	// Get extension from locals if available
	if ext := c.Locals("extension"); ext != nil {
		id = id + "." + ext.(string)
	}

	paste, err := h.services.Paste.GetPaste(id)
	if err != nil {
		return err
	}

	if err := h.services.Analytics.LogPasteView(c, paste.ID); err != nil {
		h.logger.Error("failed to log paste view", zap.Error(err))
	}

	// If the accepts header contains our vendor-specific MIME type, return the paste as JSON
	if strings.Contains(c.Get("Accept"), "application/vnd.0x45.paste+json") {
		return h.services.Paste.RenderPasteJSON(c, paste)
	}

	// If the client wants HTML (browsers), render the HTML view.
	// Specifically using "application/xhtml+xml" here since all browsers include it in their
	// Accept header, and it won't ever be automatically added as a mime type for a paste.
	if strings.Contains(c.Get("Accept"), "application/xhtml+xml") {
		return h.services.Paste.RenderPaste(c, paste)
	}

	// For all other cases, check if the client accepts the paste's mime type
	acceptHeader := c.Get("Accept")
	if acceptHeader != "" && acceptHeader != "*/*" {
		// Split accept header on commas and check if any of the accepted types match
		acceptedTypes := strings.Split(acceptHeader, ",")
		matched := false

		// Strip quality values from paste's mime type
		pasteMimeType := strings.TrimSpace(strings.Split(paste.MimeType, ";")[0])

		for _, t := range acceptedTypes {
			// Trim whitespace and remove quality values (e.g., "text/html;q=0.9")
			mediaType := strings.TrimSpace(strings.Split(t, ";")[0])
			if mediaType == pasteMimeType {
				matched = true
				break
			}
		}
		if !matched {
			return fiber.NewError(
				fiber.StatusNotAcceptable,
				fmt.Sprintf("Client accepts %s but paste has mime type %s", acceptHeader, pasteMimeType),
			)
		}
	}

	// Set content type and return raw content
	c.Set("Content-Type", paste.MimeType)
	return h.services.Paste.RenderPasteRaw(c, paste)
}

// HandleRawView serves the raw content of a paste
func (h *PasteHandlers) HandleRawView(c *fiber.Ctx) error {
	id := getPasteID(c)

	// Get extension from locals if available
	if ext := c.Locals("extension"); ext != nil {
		id = id + "." + ext.(string)
	}

	paste, err := h.services.Paste.GetPaste(id)
	if err != nil {
		return err
	}

	return h.services.Paste.RenderPasteRaw(c, paste)
}

// HandleDownload serves the content as a downloadable file
func (h *PasteHandlers) HandleDownload(c *fiber.Ctx) error {
	id := getPasteID(c)

	// Get extension from locals if available
	if ext := c.Locals("extension"); ext != nil {
		id = id + "." + ext.(string)
	}

	paste, err := h.services.Paste.GetPaste(id)
	if err != nil {
		return err
	}

	return h.services.Paste.RenderDownload(c, paste)
}

// HandleDeleteWithKey deletes a paste using its deletion key
func (h *PasteHandlers) HandleDeleteWithKey(c *fiber.Ctx) error {
	return h.services.Paste.DeleteWithKey(c, getPasteID(c))
}

// HandleListPastes returns a paginated list of pastes for the API key
func (h *PasteHandlers) HandleListPastes(c *fiber.Ctx) error {
	return h.services.Paste.ListPastes(c)
}

// HandleDeletePaste deletes a paste (requires API key ownership)
func (h *PasteHandlers) HandleDeletePaste(c *fiber.Ctx) error {
	return h.services.Paste.Delete(c, getPasteID(c))
}

// HandleUpdateExpiration updates a paste's expiration time
func (h *PasteHandlers) HandleUpdateExpiration(c *fiber.Ctx) error {
	return h.services.Paste.UpdateExpiration(c, getPasteID(c))
}

// HandleGetPasteImage returns an image of the paste suitable for Open Graph
func (h *PasteHandlers) HandleGetPasteImage(c *fiber.Ctx) error {
	id := getPasteID(c)

	// Get extension from locals if available
	if ext := c.Locals("extension"); ext != nil {
		id = id + "." + ext.(string)
	}

	paste, err := h.services.Paste.GetPaste(id)
	if err != nil {
		return err
	}

	return h.services.Paste.GetPasteImage(c, paste)
}
