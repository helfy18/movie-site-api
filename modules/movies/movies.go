package movies

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/* Converts a list of strings to a list of integers.
Returns an error otherwise */
func convertStringsToInts(strs []string)([]int, error) {
    ints := make([]int, len(strs))
    
    // Iterate over the strings
    for i, s := range strs {
        n, err := strconv.Atoi(s)
        if err != nil {
            return nil, err
        }
        ints[i] = n
    }

    return ints, nil
}

/*
	Accepts optional parameters genre, universe, exclusive,
	studio, holiday, year, director, runtime (range)
	Returns list of movies matching the description.
*/
func ListMovies(c *gin.Context) {
    var conditions []bson.M

    genres := c.QueryArray("genre")
    if len(genres) > 0 {
        genreCondition := bson.M{"$or": []bson.M{
            {"Genre": bson.M{"$in": genres}},
            {"Genre_2": bson.M{"$in": genres}},
        }}
        conditions = append(conditions, genreCondition)
    }

    universes := c.QueryArray("universe")
    if len(universes) > 0 {
        universeCondition := bson.M{"$or": []bson.M{
            {"Universe": bson.M{"$in": universes}},
            {"Sub_Universe": bson.M{"$in": universes}},
        }}
        conditions = append(conditions, universeCondition)
    }

    exclusives := c.QueryArray("exclusive")
    if len(exclusives) > 0 {
        conditions = append(conditions, bson.M{"Exclusive": bson.M{"$in": exclusives}})
    }

    studio := c.QueryArray("studio")
    if len(studio) > 0 {
        conditions = append(conditions, bson.M{"Studio": bson.M{"$in": studio}})
    }

    holiday := c.QueryArray("holiday")
    if len(holiday) > 0 {
        conditions = append(conditions, bson.M{"Holiday": bson.M{"$in": holiday}})
    }

    year := c.QueryArray("year")
    if len(year) > 0 {
        years, err := convertStringsToInts(year)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "year must be integer"})
            return
        }
        conditions = append(conditions, bson.M{"Year": bson.M{"$in": years}})
    }

    director := c.QueryArray("director")
    if len(director) > 0 {
        conditions = append(conditions, bson.M{"Director": bson.M{"$in": director}})
    }

    runtime := c.QueryArray("runtime")
    if len(runtime) > 0 {
        if len(runtime) != 2 {
            c.JSON(http.StatusBadRequest, gin.H{"error": 
                "runtime must have two values for range, start and stop"})
            return
        }
        runtimes, err := convertStringsToInts(runtime)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": 
                "runtime must have two values for range, start and stop"})
            return
        }
        sort.Ints(runtimes)
        conditions = append(conditions, bson.M{"Runtime": bson.M{"$gt": runtimes[0], "$lt": runtimes[1]}})
    }

    // Combine all conditions with $and
    var query bson.M
    if len(conditions) > 0 {
        query = bson.M{"$and": conditions}
    } else {
        query = bson.M{}
    }
    
    fmt.Println(query)
	client := c.MustGet("mongoClient").(*mongo.Client)

	collection := client.Database("jdmovies").Collection("movies")
	cursor, err := collection.Find(context.TODO(), query)
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

/*
	Accepts tmdbid(int) or (title(string) & year(int)).
    Returns information about one movie.
*/
func GetMovie(c *gin.Context) {
    query := bson.M{}
    tmdbid := c.Query("tmdbid")
    if tmdbid != "" {
        TMDBId, err := strconv.Atoi(tmdbid)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "year must be an integer"})
        }
        query["TMDBId"] = TMDBId
    } else {
        title := c.Query("title")
        year := c.Query("year")
        if (title == "" || year == "") {
            c.JSON(http.StatusBadRequest, gin.H{"error" : "Include tmdbid or title and year"})
            return
        }
        Year, err := strconv.Atoi(year)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "year must be an integer"})
        }
        query["Movie"] = title
        query["Year"] = Year
    }

	client := c.MustGet("mongoClient").(*mongo.Client)

	collection := client.Database("jdmovies").Collection("movies")

	var movie movie
	err := collection.FindOne(context.TODO(), query).Decode(&movie)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	c.IndentedJSON(http.StatusOK, movie)
}
