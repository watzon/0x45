package handlers

import "github.com/gofiber/fiber/v2"

// getPasteID extracts paste ID from request parameters
func getPasteID(c *fiber.Ctx) string {
	// First try the :id parameter
	if id := c.Params("id"); id != "" {
		return id
	}
	// Then try the path parameter
	return c.Params("*")
}
