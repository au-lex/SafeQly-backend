package handlers

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"SafeQly/internal/database"
	"SafeQly/internal/models"
	"SafeQly/internal/services"
)

var emailService *services.EmailService

// InitEmailService initializes the email service 
func InitEmailService() {
	emailService = services.NewEmailService()
}

type SignupRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	OTP         string `json:"otp" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// GenerateUserTag creates a unique tag from first name + random numbers
func GenerateUserTag(fullName string) string {
	// Extract first name
	names := strings.Fields(fullName)
	firstName := strings.ToLower(names[0])
	
	// Remove special characters and limit to first 8 characters
	cleanName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, firstName)
	
	if len(cleanName) > 8 {
		cleanName = cleanName[:8]
	}
	
	// Generate random 4-digit number
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(10000) // 0-9999
	
	// Try to create unique tag
	for i := 0; i < 100; i++ { // Try up to 100 times
		tag := fmt.Sprintf("%s%04d", cleanName, randomNum)
		
		// Check if tag exists
		var existingUser models.User
		if err := database.DB.Where("user_tag = ?", tag).First(&existingUser).Error; err == gorm.ErrRecordNotFound {
			return tag
		}
		
		// If exists, increment the number
		randomNum = (randomNum + 1) % 10000
	}
	
	// Fallback: use timestamp if all attempts fail
	return fmt.Sprintf("%s%d", cleanName, time.Now().Unix()%10000)
}

// Signup initiates user registration and sends OTP
func Signup(c *fiber.Ctx) error {
	req := new(SignupRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if user already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User with this email already exists",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Generate OTP
	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	// Delete any existing pending user with this email
	database.DB.Where("email = ?", req.Email).Delete(&models.PendingUser{})

	// Create pending user
	pendingUser := models.PendingUser{
		FullName:  req.FullName,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  string(hashedPassword),
		OTP:       otp,
		OTPExpiry: time.Now().Add(10 * time.Minute),
	}

	if err := database.DB.Create(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process signup",
		})
	}

	// Send OTP email
	if err := emailService.SendOTPEmail(req.Email, otp, "signup"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP email",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "OTP sent to your email. Please verify to complete signup.",
		"email":   req.Email,
	})
}

// VerifySignupOTP verifies OTP and creates the user account
func VerifySignupOTP(c *fiber.Ctx) error {
	req := new(VerifyOTPRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Find pending user
	var pendingUser models.PendingUser
	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
		req.Email, req.OTP, time.Now()).First(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired OTP",
		})
	}

	// Generate unique user tag
	userTag := GenerateUserTag(pendingUser.FullName)

	// Create actual user
	user := models.User{
		FullName:        pendingUser.FullName,
		Email:           pendingUser.Email,
		Phone:           pendingUser.Phone,
		Password:        pendingUser.Password,
		UserTag:         userTag,  
		Balance:         0.0,       
		IsEmailVerified: true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Delete pending user
	database.DB.Delete(&pendingUser)

	// Generate JWT token
	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Account verified and created successfully",
		"token":   token,
		"user": fiber.Map{
			"id":         user.ID,
			"full_name":  user.FullName,
			"email":      user.Email,
			"phone":      user.Phone,
			"user_tag":   user.UserTag,  // ADD THIS
			"balance":    user.Balance,   // ADD THIS
			"created_at": user.CreatedAt,
		},
	})
}

// ResendSignupOTP resends OTP for signup verification
func ResendSignupOTP(c *fiber.Ctx) error {
	req := new(ResendOTPRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Find pending user
	var pendingUser models.PendingUser
	if err := database.DB.Where("email = ?", req.Email).First(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No pending signup found for this email",
		})
	}

	// Generate new OTP
	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	// Update OTP and expiry
	pendingUser.OTP = otp
	pendingUser.OTPExpiry = time.Now().Add(10 * time.Minute)

	if err := database.DB.Save(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update OTP",
		})
	}

	// Send OTP email
	if err := emailService.SendOTPEmail(req.Email, otp, "signup"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP email",
		})
	}

	return c.JSON(fiber.Map{
		"message": "OTP resent successfully",
	})
}

// Login authenticates a user
func Login(c *fiber.Ctx) error {
	req := new(LoginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	// Check if email is verified
	if !user.IsEmailVerified {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Please verify your email before logging in",
		})
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Generate JWT token
	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
		"user": fiber.Map{
			"id":        user.ID,
			"full_name": user.FullName,
			"email":     user.Email,
			"phone":     user.Phone,
			"user_tag":  user.UserTag,  // ADD THIS
			"balance":   user.Balance,   // ADD THIS
		},
	})
}

// ForgotPassword sends OTP for password reset
func ForgotPassword(c *fiber.Ctx) error {
	req := new(ForgotPasswordRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal if email exists
		return c.JSON(fiber.Map{
			"message": "If the email exists, an OTP has been sent",
		})
	}

	// Generate OTP
	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	// Save OTP to user
	expiry := time.Now().Add(10 * time.Minute)
	user.OTP = otp
	user.OTPExpiry = &expiry

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process request",
		})
	}

	// Send OTP email
	if err := emailService.SendOTPEmail(req.Email, otp, "reset"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP email",
		})
	}

	return c.JSON(fiber.Map{
		"message": "If the email exists, an OTP has been sent",
	})
}

// ResetPassword resets password using OTP
func ResetPassword(c *fiber.Ctx) error {
	req := new(ResetPasswordRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Find user with valid OTP
	var user models.User
	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
		req.Email, req.OTP, time.Now()).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired OTP",
		})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Update password and clear OTP
	user.Password = string(hashedPassword)
	user.OTP = ""
	user.OTPExpiry = nil

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reset password",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// Helper function
func generateJWT(userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("your-secret-key")) // Use env variable in production
}