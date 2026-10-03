package movies

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// The JSON field names the frontend relies on. If a struct tag change ever
// alters this set, the test fails before the frontend finds out the hard way.
var frontendMovieFields = []string{
	"movie", "jh_score", "universe", "sub_universe", "genre", "genre_2",
	"holiday", "exclusive", "studio", "year", "review", "ranking",
	"dani_approved", "plot", "poster", "actors", "director", "directors", "ratings",
	"boxoffice", "rated", "runtime", "provider", "budget", "tmdbid",
	"recommendations", "rottentomatoes", "imdb", "metacritic", "trailer",
	"origin_country", "ms_added",
}

// A document shaped exactly like the ones scripts/syncMovies.py writes to Mongo.
var dbDocument = bson.M{
	"_id":           "ignored",
	"Movie":         "Toy Story",
	"JH_Score":      int32(95),
	"Universe":      "Pixar",
	"Sub_Universe":  "Toy Story",
	"Genre":         "Animation",
	"Genre_2":       "Comedy",
	"Holiday":       "None",
	"Exclusive":     "No",
	"Studio":        "Pixar",
	"Year":          int32(1995),
	"Review":        "A classic.",
	"Ranking":       int32(2),
	"Dani_Approved": true,
	"Plot":          "Toys come alive.",
	"Poster":        "https://image.tmdb.org/t/p/w500/uXDfjJbdP4ijW5hWSBrPrlKpxab.jpg",
	"Actors":        "Tom Hanks, Tim Allen",
	"Director":      "John Lasseter",
	"Directors":     bson.A{"John Lasseter"},
	"Ratings": bson.A{
		bson.M{"Source": "Internet Movie Database", "Value": "8.3/10"},
		bson.M{"Source": "Rotten Tomatoes", "Value": "100%"},
	},
	"BoxOffice": "394,436,586",
	"Rated":     "G",
	"Runtime":   int32(81),
	"Provider": bson.M{
		"link": "https://www.themoviedb.org/movie/862/watch?locale=CA",
		"flatrate": bson.A{
			bson.M{"logo_path": "/x.jpg", "provider_id": int32(337), "provider_name": "Disney Plus", "display_priority": int32(2)},
		},
		"rent": bson.A{
			bson.M{"logo_path": "/y.jpg", "provider_id": int32(2), "provider_name": "Apple TV", "display_priority": int32(4)},
		},
		"ads": bson.A{
			bson.M{"logo_path": "/z.jpg", "provider_id": int32(538), "provider_name": "Plex", "display_priority": int32(20)},
		},
		"free": bson.A{
			bson.M{"logo_path": "/w.jpg", "provider_id": int32(300), "provider_name": "Pluto TV", "display_priority": int32(25)},
		},
	},
	"Budget":          "30,000,000",
	"TMDBId":          int32(862),
	"Recommendations": bson.A{int32(863), int32(10193)},
	"RottenTomatoes":  "100%",
	"IMDB":            "8.3/10",
	"Metacritic":      "96/100",
	"Trailer":         "https://www.youtube.com/embed/v-PjgYDrg70",
	"origin_country":  "US",
	"ms_added":        int64(1700000000000),
}

// decodeDBDocument runs the same BSON decode path the API uses on Mongo results.
func decodeDBDocument(t *testing.T) movie {
	t.Helper()
	raw, err := bson.Marshal(dbDocument)
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}
	var m movie
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bson.Unmarshal: %v", err)
	}
	return m
}

func TestMovieDecodesFromDBDocument(t *testing.T) {
	m := decodeDBDocument(t)

	if m.Movie != "Toy Story" {
		t.Errorf("Movie = %q, want Toy Story", m.Movie)
	}
	if m.JH_Score != 95 {
		t.Errorf("JH_Score = %d, want 95", m.JH_Score)
	}
	if m.Year != 1995 {
		t.Errorf("Year = %d, want 1995", m.Year)
	}
	if m.TMDBId != 862 {
		t.Errorf("TMDBId = %d, want 862", m.TMDBId)
	}
	if m.Origin_Country != "US" {
		t.Errorf("Origin_Country = %q, want US", m.Origin_Country)
	}
	if m.Ms_added != 1700000000000 {
		t.Errorf("Ms_added = %d, want 1700000000000", m.Ms_added)
	}
	if len(m.Ratings) != 2 || m.Ratings[0].Source != "Internet Movie Database" || m.Ratings[0].Value != "8.3/10" {
		t.Errorf("Ratings decoded wrong: %+v", m.Ratings)
	}
	if len(m.Provider.Flatrate) != 1 || m.Provider.Flatrate[0].Provider_name != "Disney Plus" {
		t.Errorf("Provider.Flatrate decoded wrong: %+v", m.Provider.Flatrate)
	}
	if len(m.Provider.Rent) != 1 || m.Provider.Rent[0].Provider_id != 2 {
		t.Errorf("Provider.Rent decoded wrong: %+v", m.Provider.Rent)
	}
	if len(m.Provider.Ads) != 1 || m.Provider.Ads[0].Provider_name != "Plex" {
		t.Errorf("Provider.Ads decoded wrong: %+v", m.Provider.Ads)
	}
	if len(m.Provider.Free) != 1 || m.Provider.Free[0].Provider_name != "Pluto TV" {
		t.Errorf("Provider.Free decoded wrong: %+v", m.Provider.Free)
	}
	if !reflect.DeepEqual(m.Recommendations, []int32{863, 10193}) {
		t.Errorf("Recommendations = %v, want [863 10193]", m.Recommendations)
	}
	if !reflect.DeepEqual(m.Directors, []string{"John Lasseter"}) {
		t.Errorf("Directors = %v, want [John Lasseter]", m.Directors)
	}
	if !m.Dani_Approved {
		t.Error("Dani_Approved = false, want true")
	}
}

func TestMovieJSONMatchesFrontendContract(t *testing.T) {
	m := decodeDBDocument(t)

	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	gotFields := make([]string, 0, len(got))
	for k := range got {
		gotFields = append(gotFields, k)
	}
	sort.Strings(gotFields)
	wantFields := append([]string(nil), frontendMovieFields...)
	sort.Strings(wantFields)
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Fatalf("JSON field set changed.\ngot:  %v\nwant: %v", gotFields, wantFields)
	}

	// Spot-check values and nested shapes the frontend reads.
	if got["movie"] != "Toy Story" || got["tmdbid"] != float64(862) || got["origin_country"] != "US" {
		t.Errorf("top-level values wrong: movie=%v tmdbid=%v origin_country=%v",
			got["movie"], got["tmdbid"], got["origin_country"])
	}
	ratings := got["ratings"].([]any)
	first := ratings[0].(map[string]any)
	if first["source"] != "Internet Movie Database" || first["value"] != "8.3/10" {
		t.Errorf("ratings JSON shape wrong: %v", first)
	}
	provider := got["provider"].(map[string]any)
	for _, section := range []string{"flatrate", "ads", "free"} {
		entry := provider[section].([]any)[0].(map[string]any)
		for _, key := range []string{"logo_path", "provider_id", "provider_name", "display_priority"} {
			if _, exists := entry[key]; !exists {
				t.Errorf("provider %s entry missing %q: %v", section, key, entry)
			}
		}
	}
	if provider["link"] == "" {
		t.Error("provider link missing")
	}
}

/*
The frontend types ads and free as `ProviderInfo[] | null`: a movie without
those tiers must serialize them as null, not omit the keys.
*/
func TestProviderAdsFreeNullWhenAbsent(t *testing.T) {
	raw, err := bson.Marshal(bson.M{
		"Movie": "No Free Tier",
		"Provider": bson.M{
			"link":     "https://example.com",
			"flatrate": bson.A{},
		},
	})
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}
	var m movie
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bson.Unmarshal: %v", err)
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	provider := got["provider"].(map[string]any)
	for _, section := range []string{"ads", "free"} {
		v, exists := provider[section]
		if !exists {
			t.Errorf("provider.%s key missing; frontend expects it present (null allowed)", section)
		} else if v != nil {
			t.Errorf("provider.%s = %v, want null when absent in DB", section, v)
		}
	}
}

/*
Most real documents omit empty fields entirely (the sync script skips empty
cells): Holiday, Exclusive, Universe, Sub_Universe, Genre_2 and Review are
missing from a large share of the collection, and 27 documents have an empty
Provider. The frontend still receives every JSON key, with zero values.
*/
func TestMovieJSONContractWithSparseDocument(t *testing.T) {
	sparse := bson.M{
		"Movie":    "Obscure Film",
		"Year":     int32(1960),
		"TMDBId":   int32(999),
		"Ranking":  int32(1500),
		"Provider": bson.M{},
	}
	raw, err := bson.Marshal(sparse)
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}
	var m movie
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bson.Unmarshal: %v", err)
	}

	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, field := range frontendMovieFields {
		if _, exists := got[field]; !exists {
			t.Errorf("sparse document JSON missing key %q", field)
		}
	}
	if got["holiday"] != "" || got["universe"] != "" {
		t.Errorf("expected empty strings for omitted fields, got holiday=%v universe=%v",
			got["holiday"], got["universe"])
	}
	if got["jh_score"] != float64(0) {
		t.Errorf("expected zero jh_score, got %v", got["jh_score"])
	}
}
