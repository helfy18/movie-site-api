package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/helfy18/movie-site-api/modules/auth"
	"github.com/helfy18/movie-site-api/modules/movies"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Initialize Gin router
	router := gin.Default()

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get MongoDB URI from environment variables
	mongoURI := os.Getenv("MONGOURI")
	if mongoURI == "" {
		log.Fatal("MONGOURI environment variable not set")
	}

	// Set MongoDB client options
	opts := options.Client().ApplyURI(mongoURI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ensure connection is closed when main function exits
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// Send a ping to confirm a successful connection
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	// Middleware to inject MongoDB client into the context
	router.Use(func(c *gin.Context) {
		c.Set("mongoClient", client)
		c.Next()
	})
	
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{os.Getenv("SITEURL")}
	router.Use(cors.New(config))

	// Define routes
	router.GET("/movies/list", movies.ListMovies)
	router.GET("/movies/get", movies.GetMovie)
	router.GET("/login", auth.Login)

	// Run the Gin server
	router.Run() // Default port is 8080
}
