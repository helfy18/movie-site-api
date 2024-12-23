package movies

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
	Converts a list of strings to a list of integers.
	Returns an error otherwise
*/
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
    decade := c.QueryArray("decade")
    if len(year) > 0 || len(decade) > 0 {
        var years []int

        // Convert individual years to integers
        if len(year) > 0 {
            yearInts, err := convertStringsToInts(year)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "year must be integer"})
                return
            }
            years = append(years, yearInts...)
        }

        // Convert decades into individual years and add to the list
        for _, d := range decade {
            yearRange, err := parseDecade(d)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decade format, expected yyyy-yyyy"})
                return
            }
            years = append(years, yearRange...)
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

		provider := c.QueryArray("provider")
		if len(provider) > 0 {
			var providers []int
			providers, err := convertStringsToInts(provider)
			if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "provider must be id"})
					return
			}
			conditions = append(conditions, bson.M{"Provider.flatrate.provider_id": bson.M{"$in": providers}})
		}
		
    // Combine all conditions with $and
    var query bson.M
    if len(conditions) > 0 {
        query = bson.M{"$and": conditions}
    } else {
        query = bson.M{}
    }
    
	client := c.MustGet("mongoClient").(*mongo.Client)
	collection := client.Database("jdmovies").Collection("movies")

	findOptions := options.Find().SetSort(bson.D{{Key: "Ranking", Value: 1}})

	cursor, err := collection.Find(context.TODO(), query, findOptions)
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

// parseDecade parses a decade in the format "yyyy-yyyy" and returns a slice of individual years.
func parseDecade(decade string) ([]int, error) {
	parts := strings.Split(decade, "-")
	if len(parts) != 2 {
			return nil, fmt.Errorf("invalid decade format")
	}

	startYear, err1 := strconv.Atoi(parts[0])
	endYear, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || startYear > endYear {
			return nil, fmt.Errorf("invalid decade range")
	}

	var years []int
	for y := startYear; y <= endYear; y++ {
			years = append(years, y)
	}
	return years, nil
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

/*
	Accepts tmdbid(int[])
*/
func GetMovieById(c *gin.Context) {
	query := bson.M{}
	tmdbid := c.QueryArray("tmdbid")
	if len(tmdbid) > 0 {
		TMDBid, err := convertStringsToInts(tmdbid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error" : "tmdbid must be integer"})
		}
		query["TMDBId"] = bson.M{"$in": TMDBid}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error" : "Include at least one tmdbid"})
		return
	}
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

func ListTypes(c *gin.Context) {
	client := c.MustGet("mongoClient").(*mongo.Client)
	collection := client.Database("jdmovies").Collection("movies")

	// Aggregation universePipeline for universes and sub-universes
	universePipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id": bson.M{"Universe": "$Universe", "Sub_Universe": bson.M{"$ifNull": []interface{}{"$Sub_Universe", "__NO_SUB_UNIVERSE__"}}},
			"subUniverseCount": bson.M{"$sum": 1},
		}},
		bson.M{"$group": bson.M{
			"_id": "$_id.Universe",
			"totalCount": bson.M{"$sum": "$subUniverseCount"},
			"subUniverses": bson.M{"$push": bson.M{
				"fieldValue": "$_id.Sub_Universe",
				"totalCount": "$subUniverseCount",
			}},
		}},
		bson.M{"$project": bson.M{
			"fieldValue": "$_id",
			"totalCount": "$totalCount",
			"subUniverses": bson.M{
				"$filter": bson.M{
					"input": "$subUniverses",
					"as": "subUniverse",
					"cond": bson.M{"$ne": []interface{}{"$$subUniverse.fieldValue", "__NO_SUB_UNIVERSE__"}},
				},
			},
			"noSubUniverseCount": bson.M{
				"$sum": bson.M{
					"$map": bson.M{
						"input": "$subUniverses",
						"as": "subUniverse",
						"in": bson.M{"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$$subUniverse.fieldValue", "__NO_SUB_UNIVERSE__"}},
							"$$subUniverse.totalCount",
							0,
						}},
					},
				},
			},
		}},
	}

	cursor, err := collection.Aggregate(context.TODO(), universePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch universe data"})
		return
	}
	defer cursor.Close(context.TODO())

	var universes []bson.M
	if err := cursor.All(context.TODO(), &universes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse universe data"})
		return
	}

	genrePipeline := bson.A{
		bson.M{"$project": bson.M{
			"Genre":   "$Genre",
			"Genre_2": "$Genre_2",
		}},
		bson.M{"$facet": bson.M{
			"genre1": bson.A{
				bson.M{"$group": bson.M{
					"_id": "$Genre",
					"totalCount": bson.M{"$sum": 1},
				}},
			},
			"genre2": bson.A{
				bson.M{"$group": bson.M{
					"_id": "$Genre_2",
					"totalCount": bson.M{"$sum": 1},
				}},
			},
		}},
		bson.M{"$project": bson.M{
			"allGenres": bson.M{"$setUnion": []interface{}{"$genre1", "$genre2"}},
		}},
		bson.M{"$unwind": "$allGenres"},
		bson.M{"$group": bson.M{
			"_id": "$allGenres._id",
			"totalCount": bson.M{"$sum": "$allGenres.totalCount"},
		}},
		bson.M{"$match": bson.M{
			"_id": bson.M{"$ne": nil},
		}},
		bson.M{"$project": bson.M{
			"fieldValue": "$_id",
			"_id": 0,
			"totalCount": "$totalCount",
		}},
		bson.M{"$sort": bson.M{
			"totalCount": -1,
		}},
	}

	genreCursor, err := collection.Aggregate(context.TODO(), genrePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch genre data"})
		return
	}
	defer genreCursor.Close(context.TODO())

	var genres []bson.M
	if err := genreCursor.All(context.TODO(), &genres); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse genre data"})
		return
	}

    years, err := collection.Distinct(context.TODO(), "Year", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct years"})
		return
	}

	pipeline := bson.A{
			bson.M{"$unwind": bson.M{"path": "$Provider.flatrate"}},
			bson.M{"$group": bson.M{
					"_id": "$Provider.flatrate.provider_id",
					"logo_path": bson.M{"$first": "$Provider.flatrate.logo_path"},
					"provider_id": bson.M{"$first": "$Provider.flatrate.provider_id"},
					"provider_name": bson.M{"$first": "$Provider.flatrate.provider_name"},
					"display_priority": bson.M{"$first": "$Provider.flatrate.display_priority"},
			}},
			bson.M{"$sort": bson.M{"display_priority": 1}},
			bson.M{"$project": bson.M{
					"_id": 0,
					"logo_path": 1,
					"provider_id": 1,
					"provider_name": 1,
					"display_priority": 1,
			}},
	}

	providerCursor, err := collection.Aggregate(context.TODO(), pipeline)

	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
			return
	}
	defer providerCursor.Close(context.TODO())

	var providers []bson.M
	if err := providerCursor.All(context.TODO(), &providers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse providers"})
			return
	}
    exclusives, err := collection.Distinct(context.TODO(), "Exclusive", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct exclusives"})
		return
	}

    holidays, err := collection.Distinct(context.TODO(), "Holiday", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct holidays"})
		return
	}

    studios, err := collection.Distinct(context.TODO(), "Studio", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct studios"})
		return
	}

	directorPipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id": "$Director",
			"totalCount": bson.M{"$sum": 1},
		}},
		bson.M{"$match": bson.M{
			"totalCount": bson.M{"$gte": 3},
		}},
		bson.M{"$sort": bson.M{
			"totalCount": -1,
		}},
		bson.M{"$project": bson.M{
			"fieldValue": "$_id",
			"totalCount": 1,
			"_id": 0,
		}},
	}
	
	directorCursor, err := collection.Aggregate(context.TODO(), directorPipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch directors with counts"})
		return
	}
	defer directorCursor.Close(context.TODO())
	
	var directors []bson.M
	if err = directorCursor.All(context.TODO(), &directors); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse results"})
		return
	}

	runtimePipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id": nil,
			"max": bson.M{"$max": "$Runtime"},
			"min": bson.M{"$min": "$Runtime"},
		}},
	}

	runtimeCursor, err := collection.Aggregate(context.TODO(), runtimePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate runtimes"})
		return
	}
	defer runtimeCursor.Close(context.TODO())
	
	var runtimes []bson.M;
	if err = runtimeCursor.All(context.TODO(), &runtimes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse runtimes"})
		return
	}
		
	c.IndentedJSON(http.StatusOK, bson.M{
		"provider": providers,
		"genre":     genres,
		"year":      years,
		"exclusive": exclusives,
		"holiday":   holidays,
		"studio":    studios,
		"director":  directors,
		"universes": universes,
		"runtime": runtimes,
	})
}

func GetMovieCount(c *gin.Context) {
	client := c.MustGet("mongoClient").(*mongo.Client)
	collection := client.Database("jdmovies").Collection("movies")

	count, err := collection.CountDocuments(context.TODO(), bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count documents"})
        return
    }

    c.JSON(http.StatusOK, count)
}

func GetMostRecent(c *gin.Context) {
	client := c.MustGet("mongoClient").(*mongo.Client)
	collection := client.Database("jdmovies").Collection("movies")

	limit, err := strconv.ParseInt(c.Query("count"), 10, 64)
	if err != nil {
		limit = 20
	}

	opts := options.Find()
	opts.SetSort(bson.M{"ms_added": -1})
	opts.SetLimit(limit)

	cursor, err := collection.Find(context.TODO(), bson.M{}, opts)
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