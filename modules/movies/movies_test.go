package movies

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newTestContext returns a gin context for a GET request with the given query string.
func newTestContext(rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	return c, w
}

func TestConvertStringsToInts(t *testing.T) {
	got, err := convertStringsToInts([]string{"1", "42", "-7"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int{1, 42, -7}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := convertStringsToInts([]string{"1", "abc"}); err == nil {
		t.Error("expected error for non-integer input, got nil")
	}
}

func TestParseDecade(t *testing.T) {
	got, err := parseDecade("1990-1993")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int{1990, 1991, 1992, 1993}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, bad := range []string{"1990", "1990-1980", "199x-2000", "1990-2000-2010"} {
		if _, err := parseDecade(bad); err == nil {
			t.Errorf("parseDecade(%q): expected error, got nil", bad)
		}
	}
}

func TestBuildFilterQueryEmpty(t *testing.T) {
	c, _ := newTestContext("")
	query, ok := buildFilterQuery(c)
	if !ok {
		t.Fatal("expected ok for empty query")
	}
	if !reflect.DeepEqual(query, bson.M{}) {
		t.Errorf("expected empty filter, got %v", query)
	}
}

func TestBuildFilterQueryConditions(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bson.M
	}{
		{
			name:  "genre searches both genre fields",
			query: "genre=Action&genre=Comedy",
			want: bson.M{"$or": []bson.M{
				{"Genre": bson.M{"$in": []string{"Action", "Comedy"}}},
				{"Genre_2": bson.M{"$in": []string{"Action", "Comedy"}}},
			}},
		},
		{
			name:  "universe searches both universe fields",
			query: "universe=Marvel",
			want: bson.M{"$or": []bson.M{
				{"Universe": bson.M{"$in": []string{"Marvel"}}},
				{"Sub_Universe": bson.M{"$in": []string{"Marvel"}}},
			}},
		},
		{
			name:  "year and decade combine into one Year $in",
			query: "year=1985&decade=1990-1992",
			want:  bson.M{"Year": bson.M{"$in": []int{1985, 1990, 1991, 1992}}},
		},
		{
			name:  "runtime range is sorted",
			query: "runtime=200&runtime=100",
			want:  bson.M{"Runtime": bson.M{"$gte": 100, "$lte": 200}},
		},
		{
			name:  "rating range maps to JH_Score",
			query: "rating=90&rating=50",
			want:  bson.M{"JH_Score": bson.M{"$gte": 50, "$lte": 90}},
		},
		{
			name:  "provider maps to flatrate provider id",
			query: "provider=8",
			want:  bson.M{"Provider.flatrate.provider_id": bson.M{"$in": []int{8}}},
		},
		{
			name:  "director",
			query: "director=Brad+Bird",
			want:  bson.M{"Director": bson.M{"$in": []string{"Brad Bird"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestContext(tt.query)
			query, ok := buildFilterQuery(c)
			if !ok {
				t.Fatal("expected ok")
			}
			conditions, isAnd := query["$and"].([]bson.M)
			if !isAnd || len(conditions) != 1 {
				t.Fatalf("expected one $and condition, got %v", query)
			}
			if !reflect.DeepEqual(conditions[0], tt.want) {
				t.Errorf("got %v, want %v", conditions[0], tt.want)
			}
		})
	}
}

func TestBuildFilterQueryInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"non-integer year", "year=abc"},
		{"malformed decade", "decade=1990"},
		{"reversed decade", "decade=2000-1990"},
		{"runtime with one value", "runtime=100"},
		{"non-integer runtime", "runtime=abc&runtime=100"},
		{"rating with three values", "rating=1&rating=2&rating=3"},
		{"non-integer provider", "provider=netflix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newTestContext(tt.query)
			_, ok := buildFilterQuery(c)
			if ok {
				t.Fatal("expected ok=false for invalid input")
			}
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

// assertSingleErrorResponse checks the handler wrote exactly one well-formed
// JSON error object. Guards against the regression where a handler kept
// running after a 400 and appended a second response body.
func assertSingleErrorResponse(t *testing.T, w *httptest.ResponseRecorder, wantError string) {
	t.Helper()
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	dec := json.NewDecoder(w.Body)
	var body map[string]string
	if err := dec.Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] != wantError {
		t.Errorf("got error %q, want %q", body["error"], wantError)
	}
	if dec.More() {
		t.Error("response contains more than one JSON document (handler kept writing after the error)")
	}
}

func TestGetMovieValidation(t *testing.T) {
	t.Run("non-integer tmdbid", func(t *testing.T) {
		c, w := newTestContext("tmdbid=abc")
		GetMovie(c)
		assertSingleErrorResponse(t, w, "tmdbid must be an integer")
	})

	t.Run("missing parameters", func(t *testing.T) {
		c, w := newTestContext("")
		GetMovie(c)
		assertSingleErrorResponse(t, w, "Include tmdbid or title and year")
	})

	t.Run("non-integer year", func(t *testing.T) {
		c, w := newTestContext("title=Up&year=abc")
		GetMovie(c)
		assertSingleErrorResponse(t, w, "year must be an integer")
	})
}

func TestGetMovieByIdValidation(t *testing.T) {
	t.Run("missing tmdbid", func(t *testing.T) {
		c, w := newTestContext("")
		GetMovieById(c)
		assertSingleErrorResponse(t, w, "Include at least one tmdbid")
	})

	t.Run("non-integer tmdbid", func(t *testing.T) {
		c, w := newTestContext("tmdbid=abc")
		GetMovieById(c)
		assertSingleErrorResponse(t, w, "tmdbid must be integer")
	})
}

func TestListMoviesValidation(t *testing.T) {
	c, w := newTestContext("year=abc")
	ListMovies(c)
	assertSingleErrorResponse(t, w, "year must be integer")
}

func TestGetRandomMovieValidation(t *testing.T) {
	c, w := newTestContext("rating=high")
	GetRandomMovie(c)
	assertSingleErrorResponse(t, w, "rating must have two values for range, start and stop")
}
