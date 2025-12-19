// package services

// import (
//     "crypto/rand"
//     "fmt"
//     "log"
//     "math/big"
//     "net/smtp"
//     "os"
// )

// type EmailService struct {
//     SMTPHost string
//     SMTPPort string
//     Email    string
//     Password string
// }

// func NewEmailService() *EmailService {
//     email := os.Getenv("EMAIL_ADDRESS")
//     password := os.Getenv("EMAIL_APP_PASSWORD")
    
//     // Debug logging
//     log.Printf("📧 Email Service Initialized")
//     log.Printf("   - SMTP Host: smtp.gmail.com")
//     log.Printf("   - SMTP Port: 587")
//     log.Printf("   - Email: %s", email)
//     log.Printf("   - Password: %s", maskPassword(password))
    
//     if email == "" {
//         log.Printf("⚠️  WARNING: EMAIL_ADDRESS is empty!")
//     }
//     if password == "" {
//         log.Printf("⚠️  WARNING: EMAIL_APP_PASSWORD is empty!")
//     }
    
//     return &EmailService{
//         SMTPHost: "smtp.gmail.com",
//         SMTPPort: "587",
//         Email:    email,
//         Password: password,
//     }
// }

// // Helper function to mask password for logging
// func maskPassword(password string) string {
//     if len(password) == 0 {
//         return "❌ EMPTY"
//     }
//     if len(password) < 4 {
//         return "***"
//     }
//     return password[:2] + "****" + password[len(password)-2:]
// }

// // GenerateOTP generates a 6-digit OTP
// func GenerateOTP() (string, error) {
//     max := big.NewInt(1000000)
//     n, err := rand.Int(rand.Reader, max)
//     if err != nil {
//         return "", err
//     }
//     return fmt.Sprintf("%06d", n.Int64()), nil
// }

// // SendOTPEmail sends OTP via email
// func (es *EmailService) SendOTPEmail(to, otp, purpose string) error {
//     log.Printf("📨 Attempting to send %s OTP", purpose)
//     log.Printf("   - To: %s", to)
//     log.Printf("   - OTP: %s", otp)
//     log.Printf("   - From: %s", es.Email)
    
//     subject := "Your OTP Code"
//     var body string

//     if purpose == "signup" {
//         body = fmt.Sprintf(`
// <!DOCTYPE html>
// <html>
// <head>
//     <style>
//         body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
//         .container { max-width: 600px; margin: 0 auto; padding: 20px; }
//         .otp-box { background-color: #f4f4f4; border: 2px dashed #007bff; padding: 20px; text-align: center; margin: 20px 0; border-radius: 5px; }
//         .otp-code { font-size: 32px; font-weight: bold; color: #007bff; letter-spacing: 5px; }
//         .footer { margin-top: 30px; font-size: 12px; color: #666; }
//     </style>
// </head>
// <body>
//     <div class="container">
//         <h2>Welcome to SafeQly!</h2>
//         <p>Thank you for signing up. Please use the following OTP to verify your email address:</p>
//         <div class="otp-box">
//             <div class="otp-code">%s</div>
//         </div>
//         <p>This OTP will expire in <strong>10 minutes</strong>.</p>
//         <p>If you didn't request this, please ignore this email.</p>
//         <div class="footer">
//             <p>This is an automated message, please do not reply.</p>
//         </div>
//     </div>
// </body>
// </html>
//         `, otp)
//     } else {
//         body = fmt.Sprintf(`
// <!DOCTYPE html>
// <html>
// <head>
//     <style>
//         body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
//         .container { max-width: 600px; margin: 0 auto; padding: 20px; }
//         .otp-box { background-color: #f4f4f4; border: 2px dashed #dc3545; padding: 20px; text-align: center; margin: 20px 0; border-radius: 5px; }
//         .otp-code { font-size: 32px; font-weight: bold; color: #dc3545; letter-spacing: 5px; }
//         .footer { margin-top: 30px; font-size: 12px; color: #666; }
//     </style>
// </head>
// <body>
//     <div class="container">
//         <h2>Password Reset Request</h2>
//         <p>We received a request to reset your SafeQly account password. Use the following OTP:</p>
//         <div class="otp-box">
//             <div class="otp-code">%s</div>
//         </div>
//         <p>This OTP will expire in <strong>10 minutes</strong>.</p>
//         <p>If you didn't request this, please ignore this email and your password will remain unchanged.</p>
//         <div class="footer">
//             <p>This is an automated message, please do not reply.</p>
//         </div>
//     </div>
// </body>
// </html>
//         `, otp)
//     }

//     message := fmt.Sprintf("From: %s\r\n"+
//         "To: %s\r\n"+
//         "Subject: %s\r\n"+
//         "MIME-version: 1.0;\r\n"+
//         "Content-Type: text/html; charset=\"UTF-8\";\r\n"+
//         "\r\n"+
//         "%s\r\n", es.Email, to, subject, body)

//     log.Printf("📤 Connecting to SMTP server: %s:%s", es.SMTPHost, es.SMTPPort)
    
//     auth := smtp.PlainAuth("", es.Email, es.Password, es.SMTPHost)
//     addr := fmt.Sprintf("%s:%s", es.SMTPHost, es.SMTPPort)

//     err := smtp.SendMail(addr, auth, es.Email, []string{to}, []byte(message))
//     if err != nil {
//         log.Printf("❌ SMTP Error: %v", err)
//         return fmt.Errorf("failed to send email: %v", err)
//     }

//     log.Printf("✅ Email sent successfully to: %s", to)
//     return nil
// }















package services

import (
    "crypto/rand"
    "fmt"
    "log"
    "math/big"
    "os"

    "github.com/resend/resend-go/v2"
)

type EmailService struct {
    Client *resend.Client
    From   string
}

func NewEmailService() *EmailService {
    apiKey := os.Getenv("RESEND_API_KEY")
    fromEmail := os.Getenv("FROM_EMAIL") 
    
    // Debug logging
    log.Printf("📧 Email Service Initialized (Resend)")
    log.Printf("   - From Email: %s", fromEmail)
    log.Printf("   - API Key: %s", maskAPIKey(apiKey))
    
    if apiKey == "" {
        log.Printf("⚠️  WARNING: RESEND_API_KEY is empty!")
    }
    if fromEmail == "" {
        log.Printf("⚠️  WARNING: FROM_EMAIL is empty!")
        fromEmail = "onboarding@resend.dev" // Resend's default test email
    }
    
    client := resend.NewClient(apiKey)
    
    return &EmailService{
        Client: client,
        From:   fromEmail,
    }
}

// Helper function to mask API key for logging
func maskAPIKey(key string) string {
    if len(key) == 0 {
        return "❌ EMPTY"
    }
    if len(key) < 8 {
        return "***"
    }
    return key[:4] + "****" + key[len(key)-4:]
}

// GenerateOTP generates a 6-digit OTP
func GenerateOTP() (string, error) {
    max := big.NewInt(1000000)
    n, err := rand.Int(rand.Reader, max)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%06d", n.Int64()), nil
}

// SendOTPEmail sends OTP via email using Resend
func (es *EmailService) SendOTPEmail(to, otp, purpose string) error {
    log.Printf("📨 Attempting to send %s OTP", purpose)
    log.Printf("   - To: %s", to)
    log.Printf("   - OTP: %s", otp)
    log.Printf("   - From: %s", es.From)
    
    subject := "Your OTP Code"
    var htmlBody string

    if purpose == "signup" {
        subject = "Welcome to SafeQly - Verify Your Email"
        htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .otp-box { background-color: #f4f4f4; border: 2px dashed #007bff; padding: 20px; text-align: center; margin: 20px 0; border-radius: 5px; }
        .otp-code { font-size: 32px; font-weight: bold; color: #007bff; letter-spacing: 5px; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Welcome to SafeQly!</h2>
        <p>Thank you for signing up. Please use the following OTP to verify your email address:</p>
        <div class="otp-box">
            <div class="otp-code">%s</div>
        </div>
        <p>This OTP will expire in <strong>10 minutes</strong>.</p>
        <p>If you didn't request this, please ignore this email.</p>
        <div class="footer">
            <p>This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
        `, otp)
    } else {
        subject = "SafeQly - Password Reset Request"
        htmlBody = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .otp-box { background-color: #f4f4f4; border: 2px dashed #dc3545; padding: 20px; text-align: center; margin: 20px 0; border-radius: 5px; }
        .otp-code { font-size: 32px; font-weight: bold; color: #dc3545; letter-spacing: 5px; }
        .footer { margin-top: 30px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h2>Password Reset Request</h2>
        <p>We received a request to reset your SafeQly account password. Use the following OTP:</p>
        <div class="otp-box">
            <div class="otp-code">%s</div>
        </div>
        <p>This OTP will expire in <strong>10 minutes</strong>.</p>
        <p>If you didn't request this, please ignore this email and your password will remain unchanged.</p>
        <div class="footer">
            <p>This is an automated message, please do not reply.</p>
        </div>
    </div>
</body>
</html>
        `, otp)
    }

    // Create the email request
    params := &resend.SendEmailRequest{
        From:    es.From,
        To:      []string{to},
        Subject: subject,
        Html:    htmlBody,
    }

    log.Printf("📤 Sending email via Resend API")
    
    // Send the email
    sent, err := es.Client.Emails.Send(params)
    if err != nil {
        log.Printf("❌ Resend API Error: %v", err)
        return fmt.Errorf("failed to send email: %v", err)
    }

    log.Printf("✅ Email sent successfully to: %s (ID: %s)", to, sent.Id)
    return nil
}