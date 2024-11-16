package utils

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// QueryInt gets an integer query parameter with a default value
func QueryInt(c *fiber.Ctx, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}

	return intVal
}
