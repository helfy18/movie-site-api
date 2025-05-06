package movies

// Different websites that provide movies to watch
type providerInfo struct {
	Logo_path        string `json:"logo_path"`
	Provider_id      int32  `json:"provider_id"`
	Provider_name    string `json:"provider_name"`
	Display_priority int32  `json:"display_priority"`
}

/* The format that the providers is stored in the database.
	Contains lists of providers based on the availability of
	the movie (to rent, buy or stream) */
type providers struct {
	Link     string         `json:"link"`
	Rent     []providerInfo `json:"rent"`
	Flatrate []providerInfo `json:"flatrate"`
	Buy      []providerInfo `json:"buy"`
}

// Ratings from other sites
type rating struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}

/* Information about a movie as stored in the database. Some fields
	will be empty. */
type movie struct {
	Movie           string    `json:"movie"`
	JH_Score        int32     `json:"jh_score"`
	Universe        string    `json:"universe"`
	Sub_Universe    string    `json:"sub_universe"`
	Genre           string    `json:"genre"`
	Genre_2         string    `json:"genre_2"`
	Holiday         string    `json:"holiday"`
	Exclusive       string    `json:"exclusive"`
	Studio          string    `json:"studio"`
	Year            int32     `json:"year"`
	Review          string    `json:"review"`
	Ranking			int32	  `json:"ranking"`
	Dani_Approved	bool	  `json:"dani_approved"`
	Plot            string    `json:"plot"`
	Poster          string    `json:"poster"`
	Actors          string    `json:"actors"`
	Director        string    `json:"director"`
	Ratings         []rating  `json:"ratings"`
	BoxOffice       string    `json:"boxoffice"`
	Rated           string    `json:"rated"`
	Runtime         int32     `json:"runtime"`
	Provider        providers `json:"provider"`
	Budget          string    `json:"budget"`
	TMDBId          int32     `json:"tmdbid"`
	Recommendations []int32   `json:"recommendations"`
	RottenTomatoes	string		`json:"rottentomatoes"`
	IMDB			string		`json:"imdb"`
	Metacritic		string		`json:"metacritic"`
	Trailer			string		`json:"trailer"`
	Ms_added		int64		`json:"ms_added"`
}
