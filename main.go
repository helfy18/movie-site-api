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
	"github.com/helfy18/movie-site-api/modules/reviews"
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
	// The review endpoints send the Firebase ID token in the Authorization header
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	router.Use(cors.New(config))

	// Firebase project ID for verifying ID tokens; no service-account key needed
	firebaseProjectID := os.Getenv("FIREBASE_PROJECT_ID")
	if firebaseProjectID == "" {
		log.Fatal("FIREBASE_PROJECT_ID environment variable not set")
	}
	verifier, err := auth.NewVerifier(context.TODO(), firebaseProjectID)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase auth: %v", err)
	}

	// Database and collection names match the defaults used by scripts/syncMovies.py
	db := client.Database(getenvDefault("MONGO_DB", "jdmovies"))
	movieHandler := movies.NewHandler(db.Collection(getenvDefault("MONGO_COLLECTION", "movies")))

	// Reviews live in their own collection so the sync script's full refresh
	// of the movies collection can never touch user data.
	reviewHandler := reviews.NewHandler(db.Collection(getenvDefault("MONGO_REVIEWS_COLLECTION", "reviews")))
	if err := reviewHandler.EnsureIndexes(context.TODO()); err != nil {
		log.Fatalf("Failed to create review indexes: %v", err)
	}

	// Define routes
	router.GET("/movies/list", movieHandler.ListMovies)
	router.GET("/movies/get", movieHandler.GetMovie)
	router.GET("/movies/list/id", movieHandler.GetMovieById)
	router.GET("/types/list", movieHandler.ListTypes)
	router.GET("/movies/count", movieHandler.GetMovieCount)
	router.GET("/movies/mostRecent", movieHandler.GetMostRecent)
	router.GET("/movies/random", movieHandler.GetRandomMovie)

	router.GET("/reviews/list", reviewHandler.ListReviews)
	authed := router.Group("", auth.RequireAuth(verifier))
	authed.POST("/reviews", reviewHandler.UpsertReview)
	authed.PUT("/reviews", reviewHandler.UpsertReview)
	authed.DELETE("/reviews", reviewHandler.DeleteReview)

	// Run the Gin server (default port is 8080)
	log.Fatal(router.Run())
}
