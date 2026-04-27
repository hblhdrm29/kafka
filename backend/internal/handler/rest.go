package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/transaction-system/internal/repository"
)

type RestHandler struct {
	repo *repository.TransactionRepository
}

func NewRestHandler(repo *repository.TransactionRepository) *RestHandler {
	return &RestHandler{repo: repo}
}

type CreateTransactionRequest struct {
	Amount      float64 `json:"amount" binding:"required"`
	Description string  `json:"description" binding:"required"`
}

func (h *RestHandler) CreateTransaction(c *gin.Context) {
	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction, err := h.repo.CreateTransaction(c.Request.Context(), req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

func (h *RestHandler) GetTransactions(c *gin.Context) {
	transactions, err := h.repo.GetTransactions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}
