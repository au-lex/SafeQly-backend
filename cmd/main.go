
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "SafeQly API v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Simple routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to SafeQly API",
			"status":  "running",
		})
	})

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "SafeQly",
		})
	})

	// Start server
	port := "3000"
	log.Printf("🚀 SafeQly server starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}