package services

import (
	"encoding/json"
	"fmt"
	"SafeQly/internal/database"
	"SafeQly/internal/models"
)

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(userID uint, notifType models.NotificationType, title, message string, data map[string]interface{}) error {
	// Convert data to JSON string
	var dataJSON string
	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal notification data: %w", err)
		}
		dataJSON = string(jsonBytes)
	}

	notification := models.Notification{
		UserID:  userID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    dataJSON,
		IsRead:  false,
	}

	if err := database.DB.Create(&notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// NotifyEscrowCreated notifies seller when buyer creates an escrow
func (s *NotificationService) NotifyEscrowCreated(sellerID uint, buyerName string, amount float64, escrowID uint) error {
	return s.CreateNotification(
		sellerID,
		models.NotificationEscrowCreated,
		"New Escrow Request",
		fmt.Sprintf("%s wants to create an escrow transaction with you for ₦%.2f", buyerName, amount),
		map[string]interface{}{
			"escrow_id":  escrowID,
			"buyer_name": buyerName,
			"amount":     amount,
		},
	)
}

// NotifyEscrowAccepted notifies buyer when seller accepts
func (s *NotificationService) NotifyEscrowAccepted(buyerID uint, sellerName string, amount float64, escrowID uint) error {
	return s.CreateNotification(
		buyerID,
		models.NotificationEscrowAccepted,
		"Escrow Accepted",
		fmt.Sprintf("%s has accepted your escrow request for ₦%.2f", sellerName, amount),
		map[string]interface{}{
			"escrow_id":   escrowID,
			"seller_name": sellerName,
			"amount":      amount,
		},
	)
}

// NotifyEscrowRejected notifies buyer when seller rejects
func (s *NotificationService) NotifyEscrowRejected(buyerID uint, sellerName, reason string, amount float64, escrowID uint) error {
	return s.CreateNotification(
		buyerID,
		models.NotificationEscrowRejected,
		"Escrow Rejected",
		fmt.Sprintf("%s rejected your escrow request. Reason: %s. ₦%.2f has been refunded.", sellerName, reason, amount),
		map[string]interface{}{
			"escrow_id":   escrowID,
			"seller_name": sellerName,
			"reason":      reason,
			"amount":      amount,
		},
	)
}

// NotifyEscrowCompleted notifies buyer when seller marks as completed
func (s *NotificationService) NotifyEscrowCompleted(buyerID uint, sellerName string, amount float64, escrowID uint) error {
	return s.CreateNotification(
		buyerID,
		models.NotificationEscrowCompleted,
		"Delivery Completed",
		fmt.Sprintf("%s has marked the delivery as completed. Please review and release ₦%.2f", sellerName, amount),
		map[string]interface{}{
			"escrow_id":   escrowID,
			"seller_name": sellerName,
			"amount":      amount,
		},
	)
}

// NotifyEscrowReleased notifies seller when buyer releases funds
func (s *NotificationService) NotifyEscrowReleased(sellerID uint, buyerName string, amount float64, escrowID uint) error {
	return s.CreateNotification(
		sellerID,
		models.NotificationEscrowReleased,
		"Funds Released",
		fmt.Sprintf("%s has released ₦%.2f to your account", buyerName, amount),
		map[string]interface{}{
			"escrow_id":  escrowID,
			"buyer_name": buyerName,
			"amount":     amount,
		},
	)
}

// NotifyDisputeRaised notifies the other party when a dispute is raised
func (s *NotificationService) NotifyDisputeRaised(userID uint, raisedByName, reason string, escrowID, disputeID uint) error {
	return s.CreateNotification(
		userID,
		models.NotificationDisputeRaised,
		"Dispute Raised",
		fmt.Sprintf("%s has raised a dispute: %s", raisedByName, reason),
		map[string]interface{}{
			"escrow_id":      escrowID,
			"dispute_id":     disputeID,
			"raised_by_name": raisedByName,
			"reason":         reason,
		},
	)
}

// NotifyDisputeResolved notifies both parties when dispute is resolved
func (s *NotificationService) NotifyDisputeResolved(userID uint, winner string, resolution string, disputeID uint) error {
	title := "Dispute Resolved"
	message := fmt.Sprintf("The dispute has been resolved in favor of the %s. %s", winner, resolution)
	
	return s.CreateNotification(
		userID,
		models.NotificationDisputeResolved,
		title,
		message,
		map[string]interface{}{
			"dispute_id": disputeID,
			"winner":     winner,
			"resolution": resolution,
		},
	)
}

// NotifyDepositSuccess notifies user of successful deposit
func (s *NotificationService) NotifyDepositSuccess(userID uint, amount float64, reference string) error {
	return s.CreateNotification(
		userID,
		models.NotificationDepositSuccess,
		"Deposit Successful",
		fmt.Sprintf("Your wallet has been credited with ₦%.2f", amount),
		map[string]interface{}{
			"amount":    amount,
			"reference": reference,
		},
	)
}

// NotifyWithdrawalSuccess notifies user of successful withdrawal
func (s *NotificationService) NotifyWithdrawalSuccess(userID uint, amount float64, bankName, reference string) error {
	return s.CreateNotification(
		userID,
		models.NotificationWithdrawalSuccess,
		"Withdrawal Successful",
		fmt.Sprintf("₦%.2f has been sent to your %s account", amount, bankName),
		map[string]interface{}{
			"amount":    amount,
			"bank_name": bankName,
			"reference": reference,
		},
	)
}

// NotifyWithdrawalFailed notifies user of failed withdrawal
func (s *NotificationService) NotifyWithdrawalFailed(userID uint, amount float64, reference string) error {
	return s.CreateNotification(
		userID,
		models.NotificationWithdrawalFailed,
		"Withdrawal Failed",
		fmt.Sprintf("Your withdrawal of ₦%.2f failed and has been refunded to your wallet", amount),
		map[string]interface{}{
			"amount":    amount,
			"reference": reference,
		},
	)
}