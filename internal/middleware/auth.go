package middleware

import (
    "os"
    "strings"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

func Protected() fiber.Handler {
    return func(c *fiber.Ctx) error {
        authHeader := c.Get("Authorization")
        if authHeader == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Missing authorization header",
            })
        }

        tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
        
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
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

        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Locals("user_id", uint(claims["user_id"].(float64)))
            c.Locals("email", claims["email"].(string))
            if role, exists := claims["role"]; exists {
                c.Locals("role", role.(string))
            }
        }

        return c.Next()
    }
}

func AdminOnly() fiber.Handler {
    return func(c *fiber.Ctx) error {
        role := c.Locals("role")
        if role == nil || role.(string) != "admin" {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "Admin access required",
            })
        }
        return c.Next()
    }
}