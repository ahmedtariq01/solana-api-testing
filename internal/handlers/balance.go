package handlers

import (
	"net/http"
	"strings"

	"solana-balance-api/internal/models"
	"solana-balance-api/internal/services"
	"solana-balance-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BalanceHandler handles balance-related HTTP requests
type BalanceHandler struct {
	balanceService services.BalanceServiceInterface
}

// NewBalanceHandler creates a new BalanceHandler instance
func NewBalanceHandler(balanceService services.BalanceServiceInterface) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
	}
}

// GetBalance handles POST /api/get-balance requests
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	// Get logger with context
	log := logger.GetLogger().WithContext(c.Request.Context())

	log.Info("Processing balance request",
		zap.String("endpoint", "/api/get-balance"),
		zap.String("method", "POST"),
	)

	var req models.BalanceRequest

	// Bind JSON request without validation
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid JSON in request",
			zap.Error(err),
			zap.String("content_type", c.GetHeader("Content-Type")),
		)

		appErr := models.NewAppErrorWithDetails(
			models.ErrorCodeMalformedJSON,
			"Invalid JSON format",
			err.Error(),
		)
		models.HandleError(c, appErr, log)
		return
	}

	// Validate request data
	if len(req.Wallets) == 0 {
		log.Warn("Empty wallets array in request")

		appErr := models.NewAppErrorWithDetails(
			models.ErrorCodeEmptyWalletArray,
			"Wallets array cannot be empty",
			"At least one wallet address must be provided",
		)
		models.HandleError(c, appErr, log)
		return
	}

	log.Debug("Validating wallet addresses",
		zap.Int("wallet_count", len(req.Wallets)),
	)

	// Validate wallet addresses format
	for i, wallet := range req.Wallets {
		if !isValidSolanaAddress(wallet) {
			log.Warn("Invalid wallet address format",
				zap.String("wallet_address", wallet),
				zap.Int("wallet_index", i),
			)

			appErr := models.NewAppErrorWithDetails(
				models.ErrorCodeInvalidWallet,
				"Invalid wallet address format",
				"Wallet address: "+wallet,
			).WithContext("wallet_index", i).WithContext("wallet_address", wallet)

			models.HandleError(c, appErr, log)
			return
		}
	}

	log.Info("Fetching balances from service",
		zap.Strings("wallet_addresses", req.Wallets),
	)

	// Get balances from service
	response, err := h.balanceService.GetBalances(req.Wallets)
	if err != nil {
		log.Error("Failed to fetch balances from service",
			zap.Error(err),
			zap.Strings("wallet_addresses", req.Wallets),
		)

		appErr := models.NewAppErrorWithCause(
			models.ErrorCodeInternalError,
			"Failed to fetch balances",
			err,
		).WithContext("wallet_addresses", req.Wallets)

		models.HandleError(c, appErr, log)
		return
	}

	// Log successful response
	log.Info("Balance request completed successfully",
		zap.Int("balance_count", len(response.Balances)),
		zap.Bool("all_cached", response.Cached),
	)

	// Return successful response
	c.JSON(http.StatusOK, response)
}

// isValidSolanaAddress validates Solana wallet address format
// Solana addresses are base58 encoded and typically 32-44 characters long
func isValidSolanaAddress(address string) bool {
	// Basic validation: check length and characters
	if len(address) < 32 || len(address) > 44 {
		return false
	}

	// Check if address contains only valid base58 characters
	// Base58 alphabet: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
	validChars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	for _, char := range address {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}
