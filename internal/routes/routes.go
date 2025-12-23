package routes

import (
    "github.com/gofiber/fiber/v2"
    "SafeQly/internal/handlers"
)

func SetupRoutes(app *fiber.App) {
    // API routes
    api := app.Group("/api")

    // Auth routes
    auth := api.Group("/auth")
    
    // Signup flow with OTP
    auth.Post("/signup", handlers.Signup)                   
    auth.Post("/verify-otp", handlers.VerifySignupOTP)   
    auth.Post("/resend-otp", handlers.ResendSignupOTP)       
    
    // Login
    auth.Post("/login", handlers.Login)
    
    // Password reset flow with OTP
    auth.Post("/forgot-password", handlers.ForgotPassword)  
    auth.Post("/reset-password", handlers.ResetPassword)     

    auth.Get("/google", handlers.GoogleAuthURL)
    auth.Get("/google/callback", handlers.GoogleCallback)

    // Health check
    api.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "message": "SafeQly API v1.0",
            "status":  "running",
        })
    })
}