package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type PaystackService struct {
	SecretKey string
	BaseURL   string
}

// Paystack API Response structures
type PaystackResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type InitializePaymentResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"reference"`
	} `json:"data"`
}

type VerifyPaymentResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		ID              int64  `json:"id"`
		Domain          string `json:"domain"`
		Status          string `json:"status"`
		Reference       string `json:"reference"`
		Amount          int    `json:"amount"` // Amount in kobo (₦1 = 100 kobo)
		Message         string `json:"message"`
		GatewayResponse string `json:"gateway_response"`
		PaidAt          string `json:"paid_at"`
		CreatedAt       string `json:"created_at"`
		Channel         string `json:"channel"`
		Currency        string `json:"currency"`
		IPAddress       string `json:"ip_address"`
		Customer        struct {
			ID           int64  `json:"id"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Email        string `json:"email"`
			CustomerCode string `json:"customer_code"`
			Phone        string `json:"phone"`
		} `json:"customer"`
	} `json:"data"`
}

type TransferRecipientResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Active        bool   `json:"active"`
		CreatedAt     string `json:"createdAt"`
		Currency      string `json:"currency"`
		Domain        string `json:"domain"`
		ID            int64  `json:"id"`
		Integration   int64  `json:"integration"`
		Name          string `json:"name"`
		RecipientCode string `json:"recipient_code"`
		Type          string `json:"type"`
		UpdatedAt     string `json:"updatedAt"`
		IsDeleted     bool   `json:"is_deleted"`
		Details       struct {
			AuthorizationCode string `json:"authorization_code"`
			AccountNumber     string `json:"account_number"`
			AccountName       string `json:"account_name"`
			BankCode          string `json:"bank_code"`
			BankName          string `json:"bank_name"`
		} `json:"details"`
	} `json:"data"`
}

type InitiateTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Integration   int64  `json:"integration"`
		Domain        string `json:"domain"`
		Amount        int    `json:"amount"`
		Currency      string `json:"currency"`
		Source        string `json:"source"`
		Reason        string `json:"reason"`
		Recipient     int64  `json:"recipient"`
		Status        string `json:"status"`
		TransferCode  string `json:"transfer_code"`
		ID            int64  `json:"id"`
		CreatedAt     string `json:"createdAt"`
		UpdatedAt     string `json:"updatedAt"`
	} `json:"data"`
}

type BankListResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    []struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Code     string `json:"code"`
		Longcode string `json:"longcode"`
		Gateway  string `json:"gateway"`
		PayWithBank bool `json:"pay_with_bank"`
		Active   bool   `json:"active"`
		Country  string `json:"country"`
		Currency string `json:"currency"`
		Type     string `json:"type"`
	} `json:"data"`
}

type ResolveAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AccountNumber string `json:"account_number"`
		AccountName   string `json:"account_name"`
		BankID        int64  `json:"bank_id"`
	} `json:"data"`
}

// NewPaystackService creates a new Paystack service instance
func NewPaystackService() *PaystackService {
	return &PaystackService{
		SecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
		BaseURL:   "https://api.paystack.co",
	}
}

// makeRequest makes HTTP request to Paystack API
func (ps *PaystackService) makeRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, ps.BaseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ps.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

// InitializePayment initializes a payment transaction
func (ps *PaystackService) InitializePayment(email string, amount float64, reference string, callbackURL string) (*InitializePaymentResponse, error) {
	// Convert amount to kobo (Paystack uses kobo for NGN)
	amountInKobo := int(amount * 100)

	payload := map[string]interface{}{
		"email":        email,
		"amount":       amountInKobo,
		"reference":    reference,
		"callback_url": callbackURL,
		"currency":     "NGN",
		"metadata": map[string]string{
			"custom_fields": "SafeQly Wallet Funding",
		},
	}

	resp, err := ps.makeRequest("POST", "/transaction/initialize", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result InitializePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// VerifyPayment verifies a payment transaction
func (ps *PaystackService) VerifyPayment(reference string) (*VerifyPaymentResponse, error) {
	resp, err := ps.makeRequest("GET", "/transaction/verify/"+reference, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VerifyPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// CreateTransferRecipient creates a transfer recipient
func (ps *PaystackService) CreateTransferRecipient(accountName, accountNumber, bankCode string) (*TransferRecipientResponse, error) {
	payload := map[string]interface{}{
		"type":           "nuban",
		"name":           accountName,
		"account_number": accountNumber,
		"bank_code":      bankCode,
		"currency":       "NGN",
	}

	resp, err := ps.makeRequest("POST", "/transferrecipient", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TransferRecipientResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// InitiateTransfer initiates a transfer to a recipient
func (ps *PaystackService) InitiateTransfer(recipientCode string, amount float64, reason string, reference string) (*InitiateTransferResponse, error) {
	// Convert amount to kobo
	amountInKobo := int(amount * 100)

	payload := map[string]interface{}{
		"source":    "balance",
		"reason":    reason,
		"amount":    amountInKobo,
		"recipient": recipientCode,
		"reference": reference,
	}

	resp, err := ps.makeRequest("POST", "/transfer", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result InitiateTransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// GetBanks retrieves list of banks
func (ps *PaystackService) GetBanks(country string) (*BankListResponse, error) {
	if country == "" {
		country = "nigeria"
	}

	resp, err := ps.makeRequest("GET", "/bank?country="+country, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BankListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}

// ResolveAccountNumber verifies account number and returns account name
func (ps *PaystackService) ResolveAccountNumber(accountNumber, bankCode string) (*ResolveAccountResponse, error) {
	endpoint := fmt.Sprintf("/bank/resolve?account_number=%s&bank_code=%s", accountNumber, bankCode)

	resp, err := ps.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ResolveAccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("paystack error: %s", result.Message)
	}

	return &result, nil
}