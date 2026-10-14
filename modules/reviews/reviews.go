package reviews

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/helfy18/movie-site-api/modules/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Reviews live in their own collection, keyed by TMDBId, and are never written
to the movies collection: the sync script's full refresh replaces or deletes
movie documents wholesale, so anything stored there would be wiped.
*/
type Handler struct {
	collection *mongo.Collection
}

func NewHandler(collection *mongo.Collection) *Handler {
	return &Handler{collection: collection}
}

type review struct {
	UserId      string    `bson:"user_id" json:"user_id"`
	DisplayName string    `bson:"display_name" json:"display_name"`
	TMDBId      int       `bson:"tmdbid" json:"tmdbid"`
	Score       *int      `bson:"score,omitempty" json:"score,omitempty"`
	Text        string    `bson:"text" json:"text"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// Score shares the 0-100 scale of JH_Score so the two can sit side by side
// in the UI.
const (
	minScore    = 0
	maxScore    = 100
	maxTextRune = 2000
)

type reviewInput struct {
	TMDBId int    `json:"tmdbid"`
	Score  *int   `json:"score"`
	Text   string `json:"text"`
}

// validate returns a user-facing error message, or "" when the input is fine.
func (in *reviewInput) validate() string {
	if in.TMDBId <= 0 {
		return "tmdbid must be a positive integer"
	}
	if in.Score == nil && in.Text == "" {
		return "review must include a score or text"
	}
	if in.Score != nil && (*in.Score < minScore || *in.Score > maxScore) {
		return fmt.Sprintf("score must be between %d and %d", minScore, maxScore)
	}
	if utf8.RuneCountInString(in.Text) > maxTextRune {
		return fmt.Sprintf("text must be at most %d characters", maxTextRune)
	}
	return ""
}

/*
EnsureIndexes creates the unique (user_id, tmdbid) index that makes upserts
safe (one review per user per movie) and a tmdbid index for per-movie lookups.
Index creation is idempotent, so running it on every startup is fine.
*/
func (h *Handler) EnsureIndexes(ctx context.Context) error {
	_, err := h.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "tmdbid", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "tmdbid", Value: 1}},
		},
	})
	return err
}

/*
Accepts tmdbid(int). Public.
Returns all reviews for that movie, most recently updated first.
*/
func (h *Handler) ListReviews(c *gin.Context) {
	tmdbid, err := strconv.Atoi(c.Query("tmdbid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tmdbid must be an integer"})
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := h.collection.Find(c.Request.Context(), bson.M{"tmdbid": tmdbid}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	reviews := make([]review, 0)
	if err := cursor.All(c.Request.Context(), &reviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode reviews"})
		return
	}

	c.JSON(http.StatusOK, reviews)
}

/*
Accepts a JSON body {tmdbid(int), score(int, optional), text(string, optional)};
at least one of score and text is required. Requires authentication.
Creates the caller's review for that movie or replaces their existing one,
and returns the saved review.
*/
func (h *Handler) UpsertReview(c *gin.Context) {
	var in reviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if msg := in.validate(); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	token := auth.User(c)
	now := time.Now().UTC()

	filter := bson.M{"user_id": token.UID, "tmdbid": in.TMDBId}
	update := bson.M{
		"$set": bson.M{
			"display_name": auth.DisplayName(token),
			"score":        in.Score,
			"text":         in.Text,
			"updated_at":   now,
		},
		"$setOnInsert": bson.M{
			"user_id":    token.UID,
			"tmdbid":     in.TMDBId,
			"created_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var saved review
	if err := h.collection.FindOneAndUpdate(c.Request.Context(), filter, update, opts).Decode(&saved); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save review"})
		return
	}

	c.JSON(http.StatusOK, saved)
}

/*
Accepts tmdbid(int). Requires authentication.
Deletes the caller's review for that movie.
*/
func (h *Handler) DeleteReview(c *gin.Context) {
	tmdbid, err := strconv.Atoi(c.Query("tmdbid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tmdbid must be an integer"})
		return
	}

	token := auth.User(c)
	result := h.collection.FindOneAndDelete(c.Request.Context(), bson.M{"user_id": token.UID, "tmdbid": tmdbid})
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No review found for this movie"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
