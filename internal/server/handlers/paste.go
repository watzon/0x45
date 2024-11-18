package handlers

import (
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

	return h.services.Paste.RenderPaste(c, paste)
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

	return h.services.Paste.RenderRawContent(c, paste)
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
