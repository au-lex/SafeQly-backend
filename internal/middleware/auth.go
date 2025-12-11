
package middleware

import (
    "os"
    "strings"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

func Protected() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get token from Authorization header
        authHeader := c.Get("Authorization")
        if authHeader == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Missing authorization header",
            })
        }

        // Extract token from "Bearer <token>"
        tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
        
        // Parse and validate token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // Validate signing method
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !token.Valid {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid or expired token",
            })
        }

        // Extract claims
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Locals("user_id", uint(claims["user_id"].(float64)))
            c.Locals("email", claims["email"].(string))
        }

        return c.Next()
    }
}