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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Initialize Gin router
	router := gin.Default()

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
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	var allowOrigins []string
	for _, key := range []string{"SITEURL", "LOCALURL"} {
		if origin := os.Getenv(key); origin != "" {
			allowOrigins = append(allowOrigins, origin)
		}
	}
	if len(allowOrigins) == 0 {
		log.Fatal("SITEURL and LOCALURL environment variables not set; at least one allowed origin is required")
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = allowOrigins
	router.Use(cors.New(config))

	// Database and collection names match the defaults used by scripts/syncMovies.py
	movieHandler := movies.NewHandler(
		client.Database(getenvDefault("MONGO_DB", "jdmovies")).
			Collection(getenvDefault("MONGO_COLLECTION", "movies")))

	// Define routes
	router.GET("/movies/list", movieHandler.ListMovies)
	router.GET("/movies/get", movieHandler.GetMovie)
	router.GET("/movies/list/id", movieHandler.GetMovieById)
	router.GET("/types/list", movieHandler.ListTypes)
	router.GET("/movies/count", movieHandler.GetMovieCount)
	router.GET("/movies/mostRecent", movieHandler.GetMostRecent)
	router.GET("/movies/random", movieHandler.GetRandomMovie)

	router.POST("/auth/login", auth.Login)

	// Run the Gin server (default port is 8080)
	log.Fatal(router.Run())
}
