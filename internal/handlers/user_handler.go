// package handlers

// import (
// 	"fmt"
// 	"math/rand"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/golang-jwt/jwt/v5"
// 	"golang.org/x/crypto/bcrypt"
// 	"gorm.io/gorm"

// 	"SafeQly/internal/database"
// 	"SafeQly/internal/models"
// 	"SafeQly/internal/services"
// )

// var emailService *services.EmailService

// // InitEmailService initializes the email service 
// func InitEmailService() {
// 	emailService = services.NewEmailService()
// }

// type SignupRequest struct {
// 	FullName string `json:"full_name" validate:"required"`
// 	Email    string `json:"email" validate:"required,email"`
// 	Phone    string `json:"phone" validate:"required"`
// 	Password string `json:"password" validate:"required,min=8"`
// }

// type VerifyOTPRequest struct {
// 	Email string `json:"email" validate:"required,email"`
// 	OTP   string `json:"otp" validate:"required,len=6"`
// }

// type ResendOTPRequest struct {
// 	Email string `json:"email" validate:"required,email"`
// }

// type LoginRequest struct {
// 	Email    string `json:"email" validate:"required,email"`
// 	Password string `json:"password" validate:"required"`
// }

// type ForgotPasswordRequest struct {
// 	Email string `json:"email" validate:"required,email"`
// }

// type ResetPasswordRequest struct {
// 	Email       string `json:"email" validate:"required,email"`
// 	OTP         string `json:"otp" validate:"required,len=6"`
// 	NewPassword string `json:"new_password" validate:"required,min=8"`
// }

// // GenerateUserTag creates a unique tag from first name + random numbers
// func GenerateUserTag(fullName string) string {
// 	// Extract first name
// 	names := strings.Fields(fullName)
// 	firstName := strings.ToLower(names[0])
	
// 	// Remove special characters and limit to first 8 characters
// 	cleanName := strings.Map(func(r rune) rune {
// 		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
// 			return r
// 		}
// 		return -1
// 	}, firstName)
	
// 	if len(cleanName) > 8 {
// 		cleanName = cleanName[:8]
// 	}
	
// 	// Generate random 4-digit number
// 	rand.Seed(time.Now().UnixNano())
// 	randomNum := rand.Intn(10000) // 0-9999
	
// 	// Try to create unique tag
// 	for i := 0; i < 100; i++ { // Try up to 100 times
// 		tag := fmt.Sprintf("%s%04d", cleanName, randomNum)
		
// 		// Check if tag exists
// 		var existingUser models.User
// 		if err := database.DB.Where("user_tag = ?", tag).First(&existingUser).Error; err == gorm.ErrRecordNotFound {
// 			return tag
// 		}
		
// 		// If exists, increment the number
// 		randomNum = (randomNum + 1) % 10000
// 	}
	
// 	// Fallback: use timestamp if all attempts fail
// 	return fmt.Sprintf("%s%d", cleanName, time.Now().Unix()%10000)
// }

// // Signup initiates user registration and sends OTP
// func Signup(c *fiber.Ctx) error {
// 	req := new(SignupRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Check if user already exists
// 	var existingUser models.User
// 	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
// 		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
// 			"error": "User with this email already exists",
// 		})
// 	}

// 	// Hash password
// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to process password",
// 		})
// 	}

// 	// Generate OTP
// 	otp, err := services.GenerateOTP()
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to generate OTP",
// 		})
// 	}

// 	// Delete any existing pending user with this email
// 	database.DB.Where("email = ?", req.Email).Delete(&models.PendingUser{})

// 	// Create pending user
// 	pendingUser := models.PendingUser{
// 		FullName:  req.FullName,
// 		Email:     req.Email,
// 		Phone:     req.Phone,
// 		Password:  string(hashedPassword),
// 		OTP:       otp,
// 		OTPExpiry: time.Now().Add(10 * time.Minute),
// 	}

// 	if err := database.DB.Create(&pendingUser).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to process signup",
// 		})
// 	}

// 	// Send OTP email
// 	if err := emailService.SendOTPEmail(req.Email, otp, "signup"); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to send OTP email",
// 		})
// 	}

// 	return c.Status(fiber.StatusOK).JSON(fiber.Map{
// 		"message": "OTP sent to your email. Please verify to complete signup.",
// 		"email":   req.Email,
// 	})
// }

// // VerifySignupOTP verifies OTP and creates the user account
// func VerifySignupOTP(c *fiber.Ctx) error {
// 	req := new(VerifyOTPRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Find pending user
// 	var pendingUser models.PendingUser
// 	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
// 		req.Email, req.OTP, time.Now()).First(&pendingUser).Error; err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid or expired OTP",
// 		})
// 	}

// 	// Generate unique user tag
// 	userTag := GenerateUserTag(pendingUser.FullName)

// 	// Create actual user
// 	user := models.User{
// 		FullName:        pendingUser.FullName,
// 		Email:           pendingUser.Email,
// 		Phone:           pendingUser.Phone,
// 		Password:        pendingUser.Password,
// 		UserTag:         userTag,  
// 		Balance:         0.0,       
// 		IsEmailVerified: true,
// 	}

// 	if err := database.DB.Create(&user).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to create user",
// 		})
// 	}

// 	// Delete pending user
// 	database.DB.Delete(&pendingUser)

// 	// Generate JWT token
// 	token, err := generateJWT(user.ID, user.Email)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to generate token",
// 		})
// 	}

// 	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
// 		"message": "Account verified and created successfully",
// 		"token":   token,
// 		"user": fiber.Map{
// 			"id":         user.ID,
// 			"full_name":  user.FullName,
// 			"email":      user.Email,
// 			"phone":      user.Phone,
// 			"user_tag":   user.UserTag,
// 			"balance":    user.Balance,
// 			"created_at": user.CreatedAt,
// 		},
// 	})
// }

// // ResendSignupOTP resends OTP for signup verification
// func ResendSignupOTP(c *fiber.Ctx) error {
// 	req := new(ResendOTPRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Find pending user
// 	var pendingUser models.PendingUser
// 	if err := database.DB.Where("email = ?", req.Email).First(&pendingUser).Error; err != nil {
// 		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
// 			"error": "No pending signup found for this email",
// 		})
// 	}

// 	// Generate new OTP
// 	otp, err := services.GenerateOTP()
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to generate OTP",
// 		})
// 	}

// 	// Update OTP and expiry
// 	pendingUser.OTP = otp
// 	pendingUser.OTPExpiry = time.Now().Add(10 * time.Minute)

// 	if err := database.DB.Save(&pendingUser).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to update OTP",
// 		})
// 	}

// 	// Send OTP email
// 	if err := emailService.SendOTPEmail(req.Email, otp, "signup"); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to send OTP email",
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"message": "OTP resent successfully",
// 	})
// }

// // Login authenticates a user
// func Login(c *fiber.Ctx) error {
// 	req := new(LoginRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Find user
// 	var user models.User
// 	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
// 				"error": "Invalid email or password",
// 			})
// 		}
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Database error",
// 		})
// 	}

// 	// Check if email is verified
// 	if !user.IsEmailVerified {
// 		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
// 			"error": "Please verify your email before logging in",
// 		})
// 	}

// 	// Compare password
// 	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
// 		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
// 			"error": "Invalid email or password",
// 		})
// 	}

// 	// Generate JWT token
// 	token, err := generateJWT(user.ID, user.Email)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to generate token",
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"message": "Login successful",
// 		"token":   token,
// 		"user": fiber.Map{
// 			"id":        user.ID,
// 			"full_name": user.FullName,
// 			"email":     user.Email,
// 			"phone":     user.Phone,
// 			"user_tag":  user.UserTag,
// 			"balance":   user.Balance,
// 		},
// 	})
// }

// // ForgotPassword sends OTP for password reset
// func ForgotPassword(c *fiber.Ctx) error {
// 	req := new(ForgotPasswordRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Find user
// 	var user models.User
// 	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
// 		// Don't reveal if email exists
// 		return c.JSON(fiber.Map{
// 			"message": "If the email exists, an OTP has been sent",
// 		})
// 	}

// 	// Generate OTP
// 	otp, err := services.GenerateOTP()
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to generate OTP",
// 		})
// 	}

// 	// Save OTP to user
// 	expiry := time.Now().Add(10 * time.Minute)
// 	user.OTP = otp
// 	user.OTPExpiry = &expiry

// 	if err := database.DB.Save(&user).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to process request",
// 		})
// 	}

// 	// Send OTP email
// 	if err := emailService.SendOTPEmail(req.Email, otp, "reset"); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to send OTP email",
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"message": "If the email exists, an OTP has been sent",
// 	})
// }

// // ResetPassword resets password using OTP
// func ResetPassword(c *fiber.Ctx) error {
// 	req := new(ResetPasswordRequest)
// 	if err := c.BodyParser(req); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid request body",
// 		})
// 	}

// 	// Find user with valid OTP
// 	var user models.User
// 	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
// 		req.Email, req.OTP, time.Now()).First(&user).Error; err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "Invalid or expired OTP",
// 		})
// 	}

// 	// Hash new password
// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to process password",
// 		})
// 	}

// 	// Update password and clear OTP
// 	user.Password = string(hashedPassword)
// 	user.OTP = ""
// 	user.OTPExpiry = nil

// 	if err := database.DB.Save(&user).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to reset password",
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"message": "Password reset successfully",
// 	})
// }

// // Helper function 
// func generateJWT(userID uint, email string) (string, error) {
// 	claims := jwt.MapClaims{
// 		"user_id": userID,
// 		"email":   email,
// 		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
// 	// USE ENVIRONMENT VARIABLE - MATCHES MIDDLEWARE
// 	secret := os.Getenv("JWT_SECRET")
// 	if secret == "" {
// 		return "", fmt.Errorf("JWT_SECRET environment variable not set")
// 	}
	
// 	return token.SignedString([]byte(secret))
// }



package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
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

// Google OAuth configuration
var (
	googleOAuthURL    = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

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
	names := strings.Fields(fullName)
	firstName := strings.ToLower(names[0])
	
	cleanName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, firstName)
	
	if len(cleanName) > 8 {
		cleanName = cleanName[:8]
	}
	
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(10000)
	
	for i := 0; i < 100; i++ {
		tag := fmt.Sprintf("%s%04d", cleanName, randomNum)
		
		var existingUser models.User
		if err := database.DB.Where("user_tag = ?", tag).First(&existingUser).Error; err == gorm.ErrRecordNotFound {
			return tag
		}
		
		randomNum = (randomNum + 1) % 10000
	}
	
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

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User with this email already exists",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	database.DB.Where("email = ?", req.Email).Delete(&models.PendingUser{})

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

	var pendingUser models.PendingUser
	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
		req.Email, req.OTP, time.Now()).First(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired OTP",
		})
	}

	userTag := GenerateUserTag(pendingUser.FullName)

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

	database.DB.Delete(&pendingUser)

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
			"user_tag":   user.UserTag,
			"balance":    user.Balance,
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

	var pendingUser models.PendingUser
	if err := database.DB.Where("email = ?", req.Email).First(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No pending signup found for this email",
		})
	}

	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	pendingUser.OTP = otp
	pendingUser.OTPExpiry = time.Now().Add(10 * time.Minute)

	if err := database.DB.Save(&pendingUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update OTP",
		})
	}

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

	if !user.IsEmailVerified {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Please verify your email before logging in",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

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
			"id":              user.ID,
			"full_name":       user.FullName,
			"email":           user.Email,
			"phone":           user.Phone,
			"user_tag":        user.UserTag,
			"balance":         user.Balance,
			"profile_picture": user.ProfilePicture,
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

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return c.JSON(fiber.Map{
			"message": "If the email exists, an OTP has been sent",
		})
	}

	otp, err := services.GenerateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate OTP",
		})
	}

	expiry := time.Now().Add(10 * time.Minute)
	user.OTP = otp
	user.OTPExpiry = &expiry

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process request",
		})
	}

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

	var user models.User
	if err := database.DB.Where("email = ? AND otp = ? AND otp_expiry > ?",
		req.Email, req.OTP, time.Now()).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired OTP",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

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

// GoogleAuthURL generates the Google OAuth URL for user authorization
// GoogleAuthURL generates the Google OAuth URL for user authorization
func GoogleAuthURL(c *fiber.Ctx) error {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI") // This should be your FRONTEND URL
	
	if clientID == "" || redirectURI == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Google OAuth not configured",
		})
	}

	state := fmt.Sprintf("%d", time.Now().UnixNano())

	authURL := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=consent",
		googleOAuthURL,
		clientID,
		redirectURI,
		"openid%20email%20profile",
		state,
	)

	return c.JSON(fiber.Map{
		"auth_url": authURL,
		"state":    state,
	})
}

// GoogleCallback handles the OAuth callback - receives code from FRONTEND
func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Authorization code not provided",
			"message": "No authorization code received from Google",
		})
	}

	// Exchange code for token using the SAME redirect_uri that was used in the auth URL
	tokenResp, err := exchangeCodeForToken(code)
	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("Token exchange error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to exchange code for token",
			"message": "Could not authenticate with Google. Please try again.",
			"details": err.Error(),
		})
	}

	userInfo, err := getGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		fmt.Printf("Get user info error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get user information",
			"message": "Could not retrieve your profile from Google. Please try again.",
		})
	}

	var user models.User
	err = database.DB.Where("email = ?", userInfo.Email).First(&user).Error
	
	if err == gorm.ErrRecordNotFound {
		// New user - create account
		user, err = createGoogleUser(userInfo)
		if err != nil {
			fmt.Printf("Create user error: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to create user",
				"message": "Could not create your account. Please try again.",
			})
		}
	} else if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Database error",
			"message": "A database error occurred. Please try again.",
		})
	} else {
		// Existing user - update Google info if needed
		if user.GoogleID == "" {
			user.GoogleID = userInfo.ID
			user.ProfilePicture = userInfo.Picture
			user.IsEmailVerified = true // Mark as verified since Google verified it
			if err := database.DB.Save(&user).Error; err != nil {
				fmt.Printf("Update user error: %v\n", err)
			}
		}
	}

	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		fmt.Printf("JWT generation error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to generate token",
			"message": "Authentication successful but token generation failed. Please try logging in.",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
		"user": fiber.Map{
			"id":              user.ID,
			"full_name":       user.FullName,
			"email":           user.Email,
			"phone":           user.Phone,
			"user_tag":        user.UserTag,
			"balance":         user.Balance,
			"profile_picture": user.ProfilePicture,
		},
	})
}

// exchangeCodeForToken exchanges the authorization code for an access token
func exchangeCodeForToken(code string) (*GoogleTokenResponse, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI") // MUST match the one used in GoogleAuthURL

	if clientID == "" || clientSecret == "" || redirectURI == "" {
		return nil, fmt.Errorf("Google OAuth environment variables not properly configured")
	}

	data := fmt.Sprintf(
		"code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code, clientID, clientSecret, redirectURI,
	)

	fmt.Printf("Exchanging code with redirect_uri: %s\n", redirectURI)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(googleTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Google token exchange failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}
// getGoogleUserInfo retrieves user information from Google
func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// createGoogleUser creates a new user from Google OAuth data
func createGoogleUser(userInfo *GoogleUserInfo) (models.User, error) {
	userTag := GenerateUserTag(userInfo.Name)

	user := models.User{
		FullName:        userInfo.Name,
		Email:           userInfo.Email,
		GoogleID:        userInfo.ID,
		ProfilePicture:  userInfo.Picture,
		UserTag:         userTag,
		Balance:         0.0,
		IsEmailVerified: userInfo.VerifiedEmail,
		Password:        "",
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return models.User{}, err
	}

	return user, nil
}

// generateJWT helper function 
func generateJWT(userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable not set")
	}
	
	return token.SignedString([]byte(secret))
}