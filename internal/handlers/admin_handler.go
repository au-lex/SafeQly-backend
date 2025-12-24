package handlers

import (
    "os"
    "strconv"
    "strings"
    "time"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
    
    "SafeQly/internal/database"
    "SafeQly/internal/models"
)

type AdminHandler struct {
    db *gorm.DB
}

func NewAdminHandler() *AdminHandler {
    return &AdminHandler{
        db: database.DB,
    }
}

// AdminLogin
func (h *AdminHandler) AdminLogin(c *fiber.Ctx) error {
    var req struct {
        Email    string `json:"email" validate:"required,email"`
        Password string `json:"password" validate:"required"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Find user by email
    var user models.User
    if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid credentials",
        })
    }

    // Check if user is admin
    if !user.IsAdmin() {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }

    // Check if account is suspended
    if user.IsSuspended {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Account is suspended",
        })
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid credentials",
        })
    }

    // Generate JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "email":   user.Email,
        "role":    user.Role,
        "exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days for admin
    })

    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to generate token",
        })
    }

    return c.JSON(fiber.Map{
        "message": "Admin login successful",
        "token":   tokenString,
        "user": fiber.Map{
            "id":        user.ID,
            "full_name": user.FullName,
            "email":     user.Email,
            "role":      user.Role,
            "user_tag":  user.UserTag,
        },
    })
}

// CreateAdmin creates a new admin account ( only existing admins can create new admins)
func (h *AdminHandler) CreateAdmin(c *fiber.Ctx) error {
    var req struct {
        FullName string `json:"full_name" validate:"required"`
        Email    string `json:"email" validate:"required,email"`
        Phone    string `json:"phone" validate:"required"`
        Password string `json:"password" validate:"required,min=8"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Check if email already exists
    var existingUser models.User
    if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
        return c.Status(fiber.StatusConflict).JSON(fiber.Map{
            "error": "Email already exists",
        })
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to hash password",
        })
    }

    // Generate unique user tag
    userTag := generateUserTag(req.FullName)
    
    // Create admin user
    admin := models.User{
        FullName:        req.FullName,
        Email:           req.Email,
        Phone:           req.Phone,
        Password:        string(hashedPassword),
        UserTag:         userTag,
        Role:            "admin",
        IsEmailVerified: true, 
        Balance:         0,
        EscrowBalance:   0,
    }

    if err := h.db.Create(&admin).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create admin account",
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "Admin account created successfully",
        "admin": fiber.Map{
            "id":        admin.ID,
            "full_name": admin.FullName,
            "email":     admin.Email,
            "user_tag":  admin.UserTag,
            "role":      admin.Role,
        },
    })
}

// InitializeFirstAdmin
func (h *AdminHandler) InitializeFirstAdmin(c *fiber.Ctx) error {
    // Check if any admin already exists
    var adminCount int64
    h.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount)

    if adminCount > 0 {
        return c.Status(fiber.StatusConflict).JSON(fiber.Map{
            "error": "Admin already exists. Use the create admin endpoint with proper authorization.",
        })
    }

    var req struct {
        FullName string `json:"full_name" validate:"required"`
        Email    string `json:"email" validate:"required,email"`
        Phone    string `json:"phone" validate:"required"`
        Password string `json:"password" validate:"required,min=8"`
        SetupKey string `json:"setup_key" validate:"required"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }


    setupKey := os.Getenv("ADMIN_SETUP_KEY")
    if setupKey == "" {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Admin setup is not configured",
        })
    }

    if req.SetupKey != setupKey {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid setup key",
        })
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to hash password",
        })
    }

    // Generate unique user tag
    userTag := generateUserTag(req.FullName)

    // Create first admin
    admin := models.User{
        FullName:        req.FullName,
        Email:           req.Email,
        Phone:           req.Phone,
        Password:        string(hashedPassword),
        UserTag:         userTag,
        Role:            "admin",
        IsEmailVerified: true,
        Balance:         0,
        EscrowBalance:   0,
    }

    if err := h.db.Create(&admin).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to create admin account",
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "First admin account created successfully",
        "admin": fiber.Map{
            "id":        admin.ID,
            "full_name": admin.FullName,
            "email":     admin.Email,
            "user_tag":  admin.UserTag,
            "role":      admin.Role,
        },
    })
}

// GetAdminProfile returns the current admin's profile
func (h *AdminHandler) GetAdminProfile(c *fiber.Ctx) error {
    userID := c.Locals("user_id").(uint)

    var admin models.User
    if err := h.db.First(&admin, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Admin not found",
        })
    }

    return c.JSON(fiber.Map{
        "admin": fiber.Map{
            "id":                admin.ID,
            "full_name":         admin.FullName,
            "email":             admin.Email,
            "phone":             admin.Phone,
            "user_tag":          admin.UserTag,
            "role":              admin.Role,
            "is_email_verified": admin.IsEmailVerified,
            "created_at":        admin.CreatedAt,
        },
    })
}

// GetAllUsers retrieves all users with pagination
func (h *AdminHandler) GetAllUsers(c *fiber.Ctx) error {
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "20"))
    offset := (page - 1) * limit

    var users []models.User
    var total int64

    if err := h.db.Model(&models.User{}).Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to count users",
        })
    }

    if err := h.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to retrieve users",
        })
    }

    return c.JSON(fiber.Map{
        "users": users,
        "pagination": fiber.Map{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// GetUserByID retrieves a specific user
func (h *AdminHandler) GetUserByID(c *fiber.Ctx) error {
    userID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var user models.User
    if err := h.db.First(&user, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    return c.JSON(fiber.Map{
        "user": user,
    })
}

// UpdateUser allows admin to update user details
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
    userID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var req struct {
        Email       string `json:"email"`
        PhoneNumber string `json:"phone_number"`
        IsVerified  *bool  `json:"is_verified"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    var user models.User
    if err := h.db.First(&user, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    updates := map[string]interface{}{}
    if req.Email != "" {
        updates["email"] = req.Email
    }
    if req.PhoneNumber != "" {
        updates["phone_number"] = req.PhoneNumber
    }
    if req.IsVerified != nil {
        updates["is_verified"] = *req.IsVerified
    }

    if err := h.db.Model(&user).Updates(updates).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update user",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User updated successfully",
        "user":    user,
    })
}

// SuspendUser suspends a user account
func (h *AdminHandler) SuspendUser(c *fiber.Ctx) error {
    userID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var req struct {
        Reason string `json:"reason" validate:"required"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    var user models.User
    if err := h.db.First(&user, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    now := time.Now()
    if err := h.db.Model(&user).Updates(map[string]interface{}{
        "is_suspended":   true,
        "suspended_at":   &now,
        "suspend_reason": req.Reason,
    }).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to suspend user",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User suspended successfully",
    })
}

// UnsuspendUser reactivates a suspended user account
func (h *AdminHandler) UnsuspendUser(c *fiber.Ctx) error {
    userID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var user models.User
    if err := h.db.First(&user, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    if err := h.db.Model(&user).Updates(map[string]interface{}{
        "is_suspended":   false,
        "suspended_at":   nil,
        "suspend_reason": "",
    }).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to unsuspend user",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User unsuspended successfully",
    })
}

// DeleteUser permanently deletes a user account
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
    userID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid user ID",
        })
    }

    var user models.User
    if err := h.db.First(&user, userID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    // Check if user has active escrows
    var activeEscrows int64
    h.db.Model(&models.Escrow{}).Where("(buyer_id = ? OR seller_id = ?) AND status NOT IN (?)", 
        userID, userID, []string{"completed", "cancelled", "refunded"}).Count(&activeEscrows)

    if activeEscrows > 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Cannot delete user with active escrows",
        })
    }

    if err := h.db.Delete(&user).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to delete user",
        })
    }

    return c.JSON(fiber.Map{
        "message": "User deleted successfully",
    })
}

// GetAllTransactions retrieves all transactions with filters
func (h *AdminHandler) GetAllTransactions(c *fiber.Ctx) error {
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "20"))
    offset := (page - 1) * limit

    status := c.Query("status")
    txType := c.Query("type")

    var transactions []models.Transaction
    var total int64

    query := h.db.Model(&models.Transaction{})

    if status != "" {
        query = query.Where("status = ?", status)
    }
    if txType != "" {
        query = query.Where("type = ?", txType)
    }

    if err := query.Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to count transactions",
        })
    }

    if err := query.Preload("User").Preload("Escrow").
        Offset(offset).Limit(limit).
        Order("created_at DESC").
        Find(&transactions).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to retrieve transactions",
        })
    }

    return c.JSON(fiber.Map{
        "transactions": transactions,
        "pagination": fiber.Map{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// GetAllDisputes retrieves all disputes with filters
func (h *AdminHandler) GetAllDisputes(c *fiber.Ctx) error {
    page, _ := strconv.Atoi(c.Query("page", "1"))
    limit, _ := strconv.Atoi(c.Query("limit", "20"))
    offset := (page - 1) * limit

    status := c.Query("status")

    var disputes []models.Dispute
    var total int64

    query := h.db.Model(&models.Dispute{})

    if status != "" {
        query = query.Where("status = ?", status)
    }

    if err := query.Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to count disputes",
        })
    }

    if err := query.Preload("Escrow").Preload("Escrow.Buyer").Preload("Escrow.Seller").
        Offset(offset).Limit(limit).
        Order("created_at DESC").
        Find(&disputes).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to retrieve disputes",
        })
    }

    return c.JSON(fiber.Map{
        "disputes": disputes,
        "pagination": fiber.Map{
            "page":  page,
            "limit": limit,
            "total": total,
        },
    })
}

// GetDisputeByID retrieves a specific dispute with full details
func (h *AdminHandler) GetDisputeByID(c *fiber.Ctx) error {
    disputeID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid dispute ID",
        })
    }

    var dispute models.Dispute
    if err := h.db.Preload("Escrow").Preload("Escrow.Buyer").Preload("Escrow.Seller").
        First(&dispute, disputeID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Dispute not found",
        })
    }

    return c.JSON(fiber.Map{
        "dispute": dispute,
    })
}

// ResolveDispute resolves a dispute (admin decision)
func (h *AdminHandler) ResolveDispute(c *fiber.Ctx) error {
    disputeID, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid dispute ID",
        })
    }

    var req struct {
        Resolution string `json:"resolution" validate:"required"`
        Winner     string `json:"winner" validate:"required,oneof=buyer seller"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    var dispute models.Dispute
    if err := h.db.Preload("Escrow").First(&dispute, disputeID).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Dispute not found",
        })
    }

    if dispute.Status == "resolved" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Dispute already resolved",
        })
    }

    adminID := c.Locals("user_id").(uint)

    tx := h.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    now := time.Now()
    if err := tx.Model(&dispute).Updates(map[string]interface{}{
        "status":      "resolved",
        "resolution":  req.Resolution,
        "winner":      req.Winner,
        "resolved_at": &now,
        "resolved_by": adminID,
    }).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to resolve dispute",
        })
    }

    // Update escrow status based on resolution
    var escrowStatus string
    if req.Winner == "buyer" {
        escrowStatus = "refunded"
    } else {
        escrowStatus = "released"
    }

    if err := tx.Model(&dispute.Escrow).Update("status", escrowStatus).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update escrow status",
        })
    }

    if err := tx.Commit().Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to commit transaction",
        })
    }

    return c.JSON(fiber.Map{
        "message": "Dispute resolved successfully",
        "dispute": dispute,
    })
}

// GetDashboardStats retrieves admin dashboard statistics
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
    var stats struct {
        TotalUsers        int64 `json:"total_users"`
        ActiveUsers       int64 `json:"active_users"`
        SuspendedUsers    int64 `json:"suspended_users"`
        TotalEscrows      int64 `json:"total_escrows"`
        ActiveEscrows     int64 `json:"active_escrows"`
        CompletedEscrows  int64 `json:"completed_escrows"`
        TotalDisputes     int64 `json:"total_disputes"`
        PendingDisputes   int64 `json:"pending_disputes"`
        ResolvedDisputes  int64 `json:"resolved_disputes"`
        TotalTransactions int64 `json:"total_transactions"`
    }

    h.db.Model(&models.User{}).Count(&stats.TotalUsers)
    h.db.Model(&models.User{}).Where("is_suspended = ?", false).Count(&stats.ActiveUsers)
    h.db.Model(&models.User{}).Where("is_suspended = ?", true).Count(&stats.SuspendedUsers)
    
    h.db.Model(&models.Escrow{}).Count(&stats.TotalEscrows)
    h.db.Model(&models.Escrow{}).Where("status IN (?)", []string{"pending", "funded", "in_progress"}).Count(&stats.ActiveEscrows)
    h.db.Model(&models.Escrow{}).Where("status = ?", "completed").Count(&stats.CompletedEscrows)
    
    h.db.Model(&models.Dispute{}).Count(&stats.TotalDisputes)
    h.db.Model(&models.Dispute{}).Where("status = ?", "pending").Count(&stats.PendingDisputes)
    h.db.Model(&models.Dispute{}).Where("status = ?", "resolved").Count(&stats.ResolvedDisputes)
    
    h.db.Model(&models.Transaction{}).Count(&stats.TotalTransactions)

    return c.JSON(fiber.Map{
        "stats": stats,
    })
}





// GetPendingWithdrawals retrieves all pending withdrawals for manual processing
func (h *AdminHandler) GetPendingWithdrawals(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset := (page - 1) * limit

	var transactions []models.Transaction
	var total int64

	query := h.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionPending)

	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count pending withdrawals",
		})
	}

	if err := query.Preload("User").
		Order("created_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve pending withdrawals",
		})
	}

	// Calculate total amount pending
	var totalAmount float64
	for _, tx := range transactions {
		totalAmount += tx.Amount
	}

	return c.JSON(fiber.Map{
		"pending_withdrawals": transactions,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
		"summary": fiber.Map{
			"total_count":  total,
			"total_amount": totalAmount,
		},
		"note": "Process these manually via your bank, then mark as completed",
	})
}

// GetWithdrawalByID retrieves a specific withdrawal with user details
func (h *AdminHandler) GetWithdrawalByID(c *fiber.Ctx) error {
	txID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid transaction ID",
		})
	}

	var transaction models.Transaction
	if err := h.db.Preload("User").First(&transaction, txID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Withdrawal not found",
		})
	}

	if transaction.Type != models.TransactionWithdrawal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not a withdrawal transaction",
		})
	}

	return c.JSON(fiber.Map{
		"withdrawal": transaction,
		"user": fiber.Map{
			"id":        transaction.User.ID,
			"full_name": transaction.User.FullName,
			"email":     transaction.User.Email,
			"phone":     transaction.User.Phone,
		},
	})
}

// CompleteManualWithdrawal marks a withdrawal as completed after manual processing
func (h *AdminHandler) CompleteManualWithdrawal(c *fiber.Ctx) error {
	txID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid transaction ID",
		})
	}

	var req struct {
		Notes string `json:"notes"` // Optional admin notes
	}
	c.BodyParser(&req)

	var transaction models.Transaction
	if err := h.db.Preload("User").First(&transaction, txID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Withdrawal not found",
		})
	}

	if transaction.Type != models.TransactionWithdrawal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not a withdrawal transaction",
		})
	}

	if transaction.Status != models.TransactionPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Withdrawal is not pending",
			"status": transaction.Status,
		})
	}

	adminID := c.Locals("user_id").(uint)
	now := time.Now()

	updates := map[string]interface{}{
		"status":       models.TransactionCompleted,
		"completed_at": &now,
	}

	// Add admin notes if provided
	if req.Notes != "" {
		currentDesc := transaction.Description
		updates["description"] = currentDesc + " | Admin Notes: " + req.Notes
	}

	if err := h.db.Model(&transaction).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to complete withdrawal",
		})
	}

	// Log the action
	fmt.Printf("✅ Admin %d marked withdrawal %s as completed for user %d (₦%.2f)\n",
		adminID, transaction.Reference, transaction.UserID, transaction.Amount)

	return c.JSON(fiber.Map{
		"message": "Withdrawal marked as completed successfully",
		"withdrawal": fiber.Map{
			"id":             transaction.ID,
			"reference":      transaction.Reference,
			"amount":         transaction.Amount,
			"status":         transaction.Status,
			"user_id":        transaction.UserID,
			"account_number": transaction.AccountNumber,
			"bank_name":      transaction.BankName,
			"account_name":   transaction.AccountName,
			"completed_at":   transaction.CompletedAt,
		},
	})
}

// FailManualWithdrawal marks a withdrawal as failed and refunds the user
func (h *AdminHandler) FailManualWithdrawal(c *fiber.Ctx) error {
	txID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid transaction ID",
		})
	}

	var req struct {
		Reason string `json:"reason" validate:"required"` // Why it failed
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body. Reason is required.",
		})
	}

	var transaction models.Transaction
	if err := h.db.Preload("User").First(&transaction, txID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Withdrawal not found",
		})
	}

	if transaction.Type != models.TransactionWithdrawal {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Not a withdrawal transaction",
		})
	}

	if transaction.Status != models.TransactionPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Withdrawal is not pending",
			"status": transaction.Status,
		})
	}

	adminID := c.Locals("user_id").(uint)

	// Use database transaction to refund user
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Refund the user
		var user models.User
		if err := tx.First(&user, transaction.UserID).Error; err != nil {
			return err
		}

		user.Balance += transaction.Amount
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Update transaction status
		transaction.Status = models.TransactionFailed
		transaction.Description = transaction.Description + " | Failed: " + req.Reason

		if err := tx.Save(&transaction).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process refund",
		})
	}

	fmt.Printf("⚠️ Admin %d marked withdrawal %s as failed. ₦%.2f refunded to user %d. Reason: %s\n",
		adminID, transaction.Reference, transaction.Amount, transaction.UserID, req.Reason)

	return c.JSON(fiber.Map{
		"message": "Withdrawal marked as failed and user refunded",
		"withdrawal": fiber.Map{
			"id":        transaction.ID,
			"reference": transaction.Reference,
			"amount":    transaction.Amount,
			"status":    transaction.Status,
			"user_id":   transaction.UserID,
		},
	})
}

// GetWithdrawalStats retrieves withdrawal statistics
func (h *AdminHandler) GetWithdrawalStats(c *fiber.Ctx) error {
	var stats struct {
		TotalWithdrawals     int64   `json:"total_withdrawals"`
		PendingWithdrawals   int64   `json:"pending_withdrawals"`
		CompletedWithdrawals int64   `json:"completed_withdrawals"`
		FailedWithdrawals    int64   `json:"failed_withdrawals"`
		PendingAmount        float64 `json:"pending_amount"`
		CompletedAmount      float64 `json:"completed_amount"`
	}

	// Count statistics
	h.db.Model(&models.Transaction{}).
		Where("type = ?", models.TransactionWithdrawal).
		Count(&stats.TotalWithdrawals)

	h.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionPending).
		Count(&stats.PendingWithdrawals)

	h.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionCompleted).
		Count(&stats.CompletedWithdrawals)

	h.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionFailed).
		Count(&stats.FailedWithdrawals)

	// Calculate amounts
	var pendingTxs []models.Transaction
	h.db.Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionPending).
		Find(&pendingTxs)
	for _, tx := range pendingTxs {
		stats.PendingAmount += tx.Amount
	}

	var completedTxs []models.Transaction
	h.db.Where("type = ? AND status = ?", models.TransactionWithdrawal, models.TransactionCompleted).
		Find(&completedTxs)
	for _, tx := range completedTxs {
		stats.CompletedAmount += tx.Amount
	}

	return c.JSON(fiber.Map{
		"stats": stats,
	})
}

// Helper function to generate user tag
func generateUserTag(fullName string) string {
    tag := strings.ToLower(strings.ReplaceAll(fullName, " ", ""))
    timestamp := time.Now().Unix()
    return tag + "_" + strconv.FormatInt(timestamp%10000, 10)
}