package movies

// Different websites that provide movies to watch
type providerInfo struct {
	Logo_path        string `json:"logo_path" bson:"logo_path"`
	Provider_id      int32  `json:"provider_id" bson:"provider_id"`
	Provider_name    string `json:"provider_name" bson:"provider_name"`
	Display_priority int32  `json:"display_priority" bson:"display_priority"`
}

/*
	 The format that the providers is stored in the database.
		Contains lists of providers based on the availability of
		the movie (to rent, buy or stream)
*/
type providers struct {
	Link     string         `json:"link" bson:"link"`
	Rent     []providerInfo `json:"rent" bson:"rent"`
	Flatrate []providerInfo `json:"flatrate" bson:"flatrate"`
	Buy      []providerInfo `json:"buy" bson:"buy"`
	Ads      []providerInfo `json:"ads" bson:"ads"`
	Free     []providerInfo `json:"free" bson:"free"`
}

// Ratings from other sites
type rating struct {
	Source string `json:"source" bson:"Source"`
	Value  string `json:"value" bson:"Value"`
}

/*
	 Information about a movie as stored in the database. Some fields
		will be empty. The bson tags must match the sheet column names
		written by scripts/syncMovies.py.
*/
type movie struct {
	Movie           string    `json:"movie" bson:"Movie"`
	JH_Score        int32     `json:"jh_score" bson:"JH_Score"`
	Universe        string    `json:"universe" bson:"Universe"`
	Sub_Universe    string    `json:"sub_universe" bson:"Sub_Universe"`
	Genre           string    `json:"genre" bson:"Genre"`
	Genre_2         string    `json:"genre_2" bson:"Genre_2"`
	Holiday         string    `json:"holiday" bson:"Holiday"`
	Exclusive       string    `json:"exclusive" bson:"Exclusive"`
	Studio          string    `json:"studio" bson:"Studio"`
	Year            int32     `json:"year" bson:"Year"`
	Review          string    `json:"review" bson:"Review"`
	Ranking         int32     `json:"ranking" bson:"Ranking"`
	Dani_Approved   bool      `json:"dani_approved" bson:"Dani_Approved"`
	Plot            string    `json:"plot" bson:"Plot"`
	Poster          string    `json:"poster" bson:"Poster"`
	Actors          string    `json:"actors" bson:"Actors"`
	Director        string    `json:"director" bson:"Director"`
	Directors       []string  `json:"directors" bson:"Directors"`
	Ratings         []rating  `json:"ratings" bson:"Ratings"`
	BoxOffice       string    `json:"boxoffice" bson:"BoxOffice"`
	Rated           string    `json:"rated" bson:"Rated"`
	Runtime         int32     `json:"runtime" bson:"Runtime"`
	Provider        providers `json:"provider" bson:"Provider"`
	Budget          string    `json:"budget" bson:"Budget"`
	TMDBId          int32     `json:"tmdbid" bson:"TMDBId"`
	Recommendations []int32   `json:"recommendations" bson:"Recommendations"`
	RottenTomatoes  string    `json:"rottentomatoes" bson:"RottenTomatoes"`
	IMDB            string    `json:"imdb" bson:"IMDB"`
	Metacritic      string    `json:"metacritic" bson:"Metacritic"`
	Trailer         string    `json:"trailer" bson:"Trailer"`
	Origin_Country  string    `json:"origin_country" bson:"origin_country"`
	Ms_added        int64     `json:"ms_added" bson:"ms_added"`
}
