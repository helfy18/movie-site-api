package movies

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type providerInfo struct {
	Logo_path string `json:"logo_path"`
	Provider_id int32 `json:"provider_id"`
	Provider_name string `json:"provider_name"`
	Display_priority int32 `json:"display_priority"`
}

type providers struct {
	Link string `json:"link"`
	Rent []providerInfo `json:"rent"`
	Flatrate []providerInfo `json:"flatrate"`
	Buy []providerInfo `json:"buy"`
}

type rating struct {
	Source string `json:"source"`
	Value string `json:"value"`
}

// album represents data about a record album.
type movie struct {
	Movie  string  `json:"movie"`
	JH_Score  int32  `json:"jh_score"`
	JV_Score int32 `json:"jv_score"`
	Universe string  `json:"universe"`
	Sub_Universe  string `json:"sub_universe"`
    Genre string `json:"genre"`
    Genre_2 string `json:"genre_2"`
    Holiday string `json:"holiday"`
    Exclusive string `json:"exclusive"`
    Studio string `json:"studio"`
    Year int32 `json:"year"`
    Review string `json:"review"`
    Plot string `json:"plot"`
    Poster string `json:"poster"`
    Actors string `json:"actors"`
    Director string `json:"director"`
    Ratings []rating `json:"ratings"`
    BoxOffice string `json:"boxoffice"`
    Rated string `json:"rated"`
    Runtime int32 `json:"runtime"`
    Provider providers `json:"provider"` //could be json?
    Budget string `json:"budget"`
    TMDBId int32 `json:"tmdbid"`
    Recommendations []int32 `json:"recommendations"`
}

// GetAlbums responds with the list of all albums as JSON.
func GetMovies(c *gin.Context) {
	// Fetch movies from MongoDB (example, assuming you have a MongoDB client passed)
	client := c.MustGet("mongoClient").(*mongo.Client)

	collection := client.Database("jdmovies").Collection("movies")
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	var movies []movie
	if err := cursor.All(context.TODO(), &movies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies " + err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, movies)
}
