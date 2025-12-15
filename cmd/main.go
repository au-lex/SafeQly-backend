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
		log.Println("⚠️  No .env file found, using environment variables")
	}

	// DEBUG: Print loaded env variables (remove in production)
	log.Printf("🔍 DEBUG - Environment Variables:")
	log.Printf("   EMAIL_ADDRESS: '%s'", os.Getenv("EMAIL_ADDRESS"))
	log.Printf("   EMAIL_APP_PASSWORD: '%s'", maskPassword(os.Getenv("EMAIL_APP_PASSWORD")))
	log.Printf("   DB_HOST: '%s'", os.Getenv("DB_HOST"))
	log.Printf("   JWT_SECRET: '%s'", maskPassword(os.Getenv("JWT_SECRET")))
	log.Printf("   PAYSTACK_SECRET_KEY: '%s'", maskPassword(os.Getenv("PAYSTACK_SECRET_KEY")))
	log.Printf("   CLOUDINARY_CLOUD_NAME: '%s'", os.Getenv("CLOUDINARY_CLOUD_NAME"))
	log.Printf("   ADMIN_SETUP_KEY: '%s'", maskPassword(os.Getenv("ADMIN_SETUP_KEY")))

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("❌ Failed to migrate database:", err)
	}
	log.Println("✅ Database connected and migrated successfully")

	// Initialize services
	handlers.InitEmailService()
	handlers.InitPaystackService()

	// Initialize Cloudinary service
	if err := handlers.InitCloudinaryService(); err != nil {
		log.Fatal("❌ Failed to initialize Cloudinary service:", err)
	}
	log.Println("✅ Cloudinary service initialized successfully")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:   "SafeQly API v1.0",
		BodyLimit: 10 * 1024 * 1024, 
	})

	// Middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Health check routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to SafeQly API",
			"status":  "running",
			"version": "1.0",
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
	routes.SetupWalletRoutes(app)
	routes.SetupEscrowRoutes(app)
	routes.SetupDisputeRoutes(app)
	routes.SetupAdminRoutes(app) 

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 SafeQly server starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}

// Helper function to mask sensitive data in logs
func maskPassword(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}