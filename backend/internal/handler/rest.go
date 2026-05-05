package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
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

type CreateProductRequest struct {
	Name  string  `json:"name" binding:"required"`
	Price float64 `json:"price" binding:"required"`
	Stock int     `json:"stock"`
}

func publishToRabbitMQ(payload []byte, eventType string) {
	rmqUser := os.Getenv("RABBITMQ_USER")
	rmqPass := os.Getenv("RABBITMQ_PASS")
	rmqHost := os.Getenv("RABBITMQ_HOST")
	rmqPort := os.Getenv("RABBITMQ_PORT")

	if rmqUser == "" {
		rmqUser = "guest"
	}
	if rmqPass == "" {
		rmqPass = "guest"
	}
	if rmqHost == "" {
		rmqHost = "127.0.0.1"
	}
	if rmqPort == "" {
		rmqPort = "5672"
	}

	rmqURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rmqUser, rmqPass, rmqHost, rmqPort)
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		log.Println("Failed to connect to RabbitMQ:", err)
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Println("Failed to open RabbitMQ channel:", err)
		return
	}
	defer ch.Close()

	ch.ExchangeDeclare("transaction_exchange", "fanout", true, false, false, false, nil)
	ch.Publish("transaction_exchange", "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
		Type:        eventType,
	})
	log.Printf("Published %s to RabbitMQ\n", eventType)
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

	// Broadcast ke RabbitMQ agar Web 1 & Web 2 juga dapat datanya
	payload, _ := json.Marshal(transaction)
	go publishToRabbitMQ(payload, "TRANSACTION_CREATED")

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

func (h *RestHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.repo.CreateProduct(c.Request.Context(), req.Name, req.Price, req.Stock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func (h *RestHandler) GetProducts(c *gin.Context) {
	products, err := h.repo.GetProducts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, products)
}
