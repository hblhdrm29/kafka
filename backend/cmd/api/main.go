package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/user/transaction-system/internal/database"
	"github.com/user/transaction-system/internal/handler"
	"github.com/user/transaction-system/internal/repository"
)

func main() {
	godotenv.Load(".env")
	godotenv.Load("../.env")
	godotenv.Load("../../.env")
	godotenv.Load("../../../.env")

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "user:password@tcp(localhost:3306)/transactions_db?parseTime=true"
	}

	// 1. Database & Repository
	db, err := database.NewMySQLConnection(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewTransactionRepository(db)

	// 2. Handlers
	restHandler := handler.NewRestHandler(repo)

	// 3. Gin Router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/transactions", restHandler.CreateTransaction)
		api.GET("/transactions", restHandler.GetTransactions)
		api.GET("/products", restHandler.GetProducts)
		api.POST("/products", restHandler.CreateProduct)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}
