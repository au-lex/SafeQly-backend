package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	
	"SafeQly/internal/database"
	"SafeQly/internal/handlers" 
	"SafeQly/internal/routes"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// DEBUG: Print loaded env variables
	log.Printf("🔍 DEBUG - Environment Variables:")
	log.Printf("   EMAIL_ADDRESS: '%s'", os.Getenv("EMAIL_ADDRESS"))
	log.Printf("   EMAIL_APP_PASSWORD: '%s'", os.Getenv("EMAIL_APP_PASSWORD"))
	log.Printf("   DB_HOST: '%s'", os.Getenv("DB_HOST"))

	// Initialize email service
	handlers.InitEmailService()

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("✓ Database connected and migrated successfully")

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
			"status":  "ok",
			"service": "SafeQly",
		})
	})

	// Setup application routes
	routes.SetupRoutes(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 SafeQly server starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}