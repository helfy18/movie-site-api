package movies

import (
	"errors"
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
Handler holds the shared dependencies for the movie endpoints. The collection
is injected once at startup instead of being passed through gin's context on
every request.
*/
type Handler struct {
	collection *mongo.Collection
}

func NewHandler(collection *mongo.Collection) *Handler {
	return &Handler{collection: collection}
}

/*
Converts a list of strings to a list of integers.
Returns an error otherwise
*/
func convertStringsToInts(strs []string) ([]int, error) {
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
Builds a MongoDB filter from the optional query parameters genre, universe,
exclusive, studio, holiday, year, decade, director, runtime (range),
rating (range) and provider. Writes a 400 response and returns false
when a parameter is invalid.
*/
func buildFilterQuery(c *gin.Context) (bson.M, bool) {
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
				return nil, false
			}
			years = append(years, yearInts...)
		}

		// Convert decades into individual years and add to the list
		for _, d := range decade {
			yearRange, err := parseDecade(d)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decade format, expected yyyy-yyyy"})
				return nil, false
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "runtime must have two values for range, start and stop"})
			return nil, false
		}
		runtimes, err := convertStringsToInts(runtime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "runtime must have two values for range, start and stop"})
			return nil, false
		}
		sort.Ints(runtimes)
		conditions = append(conditions, bson.M{"Runtime": bson.M{"$gte": runtimes[0], "$lte": runtimes[1]}})
	}

	rating := c.QueryArray("rating")
	if len(rating) > 0 {
		if len(rating) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rating must have two values for range, start and stop"})
			return nil, false
		}
		ratings, err := convertStringsToInts(rating)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rating must have two values for range, start and stop"})
			return nil, false
		}
		sort.Ints(ratings)
		conditions = append(conditions, bson.M{"JH_Score": bson.M{"$gte": ratings[0], "$lte": ratings[1]}})
	}

	provider := c.QueryArray("provider")
	if len(provider) > 0 {
		providers, err := convertStringsToInts(provider)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider must be id"})
			return nil, false
		}
		conditions = append(conditions, bson.M{"Provider.flatrate.provider_id": bson.M{"$in": providers}})
	}

	// Combine all conditions with $and
	if len(conditions) > 0 {
		return bson.M{"$and": conditions}, true
	}
	return bson.M{}, true
}

/*
Accepts optional parameters genre, universe, exclusive, studio, holiday,
year, decade, director, runtime (range), rating (range) and provider.
Returns list of movies matching the description.
*/
func (h *Handler) ListMovies(c *gin.Context) {
	query, ok := buildFilterQuery(c)
	if !ok {
		return
	}


	findOptions := options.Find().SetSort(bson.D{{Key: "Ranking", Value: 1}})

	cursor, err := h.collection.Find(c.Request.Context(), query, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	movies := make([]movie, 0)
	if err := cursor.All(c.Request.Context(), &movies); err != nil {
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
func (h *Handler) GetMovie(c *gin.Context) {
	query := bson.M{}
	tmdbid := c.Query("tmdbid")
	if tmdbid != "" {
		TMDBId, err := strconv.Atoi(tmdbid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tmdbid must be an integer"})
			return
		}
		query["TMDBId"] = TMDBId
	} else {
		title := c.Query("title")
		year := c.Query("year")
		if title == "" || year == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Include tmdbid or title and year"})
			return
		}
		Year, err := strconv.Atoi(year)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "year must be an integer"})
			return
		}
		query["Movie"] = title
		query["Year"] = Year
	}


	var movie movie
	err := h.collection.FindOne(c.Request.Context(), query).Decode(&movie)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No movie found matching the criteria"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	c.IndentedJSON(http.StatusOK, movie)
}

/*
Accepts tmdbid(int[])
*/
func (h *Handler) GetMovieById(c *gin.Context) {
	query := bson.M{}
	tmdbid := c.QueryArray("tmdbid")
	if len(tmdbid) > 0 {
		TMDBid, err := convertStringsToInts(tmdbid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tmdbid must be integer"})
			return
		}
		query["TMDBId"] = bson.M{"$in": TMDBid}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Include at least one tmdbid"})
		return
	}

	cursor, err := h.collection.Find(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}
	movies := make([]movie, 0)

	if err := cursor.All(c.Request.Context(), &movies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies " + err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, movies)
}

/*
Accepts the same optional filter parameters as ListMovies.
Returns one random movie matching the description.
*/
func (h *Handler) GetRandomMovie(c *gin.Context) {
	query, ok := buildFilterQuery(c)
	if !ok {
		return
	}

	pipeline := bson.A{
		bson.M{"$match": query},
		bson.M{"$sample": bson.M{"size": 1}},
	}


	cursor, err := h.collection.Aggregate(c.Request.Context(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}
	var movies []movie
	if err := cursor.All(c.Request.Context(), &movies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movie " + err.Error()})
		return
	}

	if len(movies) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No movies found matching the criteria"})
		return
	}

	c.IndentedJSON(http.StatusOK, movies[0])
}

func (h *Handler) ListTypes(c *gin.Context) {

	// Aggregation universePipeline for universes and sub-universes
	universePipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":              bson.M{"Universe": "$Universe", "Sub_Universe": bson.M{"$ifNull": []interface{}{"$Sub_Universe", "__NO_SUB_UNIVERSE__"}}},
			"subUniverseCount": bson.M{"$sum": 1},
		}},
		bson.M{"$group": bson.M{
			"_id":        "$_id.Universe",
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
					"as":    "subUniverse",
					"cond":  bson.M{"$ne": []interface{}{"$$subUniverse.fieldValue", "__NO_SUB_UNIVERSE__"}},
				},
			},
			"noSubUniverseCount": bson.M{
				"$sum": bson.M{
					"$map": bson.M{
						"input": "$subUniverses",
						"as":    "subUniverse",
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

	cursor, err := h.collection.Aggregate(c.Request.Context(), universePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch universe data"})
		return
	}
	defer cursor.Close(c.Request.Context())

	var universes []bson.M
	if err := cursor.All(c.Request.Context(), &universes); err != nil {
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
					"_id":        "$Genre",
					"totalCount": bson.M{"$sum": 1},
				}},
			},
			"genre2": bson.A{
				bson.M{"$group": bson.M{
					"_id":        "$Genre_2",
					"totalCount": bson.M{"$sum": 1},
				}},
			},
		}},
		bson.M{"$project": bson.M{
			"allGenres": bson.M{"$setUnion": []interface{}{"$genre1", "$genre2"}},
		}},
		bson.M{"$unwind": "$allGenres"},
		bson.M{"$group": bson.M{
			"_id":        "$allGenres._id",
			"totalCount": bson.M{"$sum": "$allGenres.totalCount"},
		}},
		bson.M{"$match": bson.M{
			"_id": bson.M{"$ne": nil},
		}},
		bson.M{"$project": bson.M{
			"fieldValue": "$_id",
			"_id":        0,
			"totalCount": "$totalCount",
		}},
		bson.M{"$sort": bson.M{
			"totalCount": -1,
		}},
	}

	genreCursor, err := h.collection.Aggregate(c.Request.Context(), genrePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch genre data"})
		return
	}
	defer genreCursor.Close(c.Request.Context())

	var genres []bson.M
	if err := genreCursor.All(c.Request.Context(), &genres); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse genre data"})
		return
	}

	years, err := h.collection.Distinct(c.Request.Context(), "Year", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct years"})
		return
	}

	pipeline := bson.A{
		bson.M{"$unwind": bson.M{"path": "$Provider.flatrate"}},
		bson.M{"$group": bson.M{
			"_id":              "$Provider.flatrate.provider_id",
			"logo_path":        bson.M{"$first": "$Provider.flatrate.logo_path"},
			"provider_id":      bson.M{"$first": "$Provider.flatrate.provider_id"},
			"provider_name":    bson.M{"$first": "$Provider.flatrate.provider_name"},
			"display_priority": bson.M{"$first": "$Provider.flatrate.display_priority"},
		}},
		bson.M{"$sort": bson.M{"display_priority": 1}},
		bson.M{"$project": bson.M{
			"_id":              0,
			"logo_path":        1,
			"provider_id":      1,
			"provider_name":    1,
			"display_priority": 1,
		}},
	}

	providerCursor, err := h.collection.Aggregate(c.Request.Context(), pipeline)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
		return
	}
	defer providerCursor.Close(c.Request.Context())

	var providers []bson.M
	if err := providerCursor.All(c.Request.Context(), &providers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse providers"})
		return
	}
	exclusives, err := h.collection.Distinct(c.Request.Context(), "Exclusive", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct exclusives"})
		return
	}

	holidays, err := h.collection.Distinct(c.Request.Context(), "Holiday", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct holidays"})
		return
	}

	studios, err := h.collection.Distinct(c.Request.Context(), "Studio", bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distinct studios"})
		return
	}

	directorPipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":        "$Director",
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
			"_id":        0,
		}},
	}

	directorCursor, err := h.collection.Aggregate(c.Request.Context(), directorPipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch directors with counts"})
		return
	}
	defer directorCursor.Close(c.Request.Context())

	var directors []bson.M
	if err = directorCursor.All(c.Request.Context(), &directors); err != nil {
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

	runtimeCursor, err := h.collection.Aggregate(c.Request.Context(), runtimePipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate runtimes"})
		return
	}
	defer runtimeCursor.Close(c.Request.Context())

	var runtimes []bson.M
	if err = runtimeCursor.All(c.Request.Context(), &runtimes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse runtimes"})
		return
	}

	c.IndentedJSON(http.StatusOK, bson.M{
		"provider":  providers,
		"genre":     genres,
		"year":      years,
		"exclusive": exclusives,
		"holiday":   holidays,
		"studio":    studios,
		"director":  directors,
		"universes": universes,
		"runtime":   runtimes,
	})
}

func (h *Handler) GetMovieCount(c *gin.Context) {

	count, err := h.collection.CountDocuments(c.Request.Context(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count documents"})
		return
	}

	c.JSON(http.StatusOK, count)
}

func (h *Handler) GetMostRecent(c *gin.Context) {
	const defaultLimit, maxLimit = 20, 100
	limit := int64(defaultLimit)
	if count := c.Query("count"); count != "" {
		parsed, err := strconv.ParseInt(count, 10, 64)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "count must be a positive integer"})
			return
		}
		limit = min(parsed, maxLimit)
	}


	opts := options.Find()
	opts.SetSort(bson.M{"ms_added": -1})
	opts.SetLimit(limit)

	cursor, err := h.collection.Find(c.Request.Context(), bson.M{}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}
	movies := make([]movie, 0)

	if err := cursor.All(c.Request.Context(), &movies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies " + err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, movies)
}
