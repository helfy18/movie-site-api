package movies

type providerInfo struct {
	Logo_path        string `json:"logo_path"`
	Provider_id      int32  `json:"provider_id"`
	Provider_name    string `json:"provider_name"`
	Display_priority int32  `json:"display_priority"`
}

type providers struct {
	Link     string         `json:"link"`
	Rent     []providerInfo `json:"rent"`
	Flatrate []providerInfo `json:"flatrate"`
	Buy      []providerInfo `json:"buy"`
}

type rating struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}

type movie struct {
	Movie           string    `json:"movie"`
	JH_Score        int32     `json:"jh_score"`
	JV_Score        int32     `json:"jv_score"`
	Universe        string    `json:"universe"`
	Sub_Universe    string    `json:"sub_universe"`
	Genre           string    `json:"genre"`
	Genre_2         string    `json:"genre_2"`
	Holiday         string    `json:"holiday"`
	Exclusive       string    `json:"exclusive"`
	Studio          string    `json:"studio"`
	Year            int32     `json:"year"`
	Review          string    `json:"review"`
	Plot            string    `json:"plot"`
	Poster          string    `json:"poster"`
	Actors          string    `json:"actors"`
	Director        string    `json:"director"`
	Ratings         []rating  `json:"ratings"`
	BoxOffice       string    `json:"boxoffice"`
	Rated           string    `json:"rated"`
	Runtime         int32     `json:"runtime"`
	Provider        providers `json:"provider"` //could be json?
	Budget          string    `json:"budget"`
	TMDBId          int32     `json:"tmdbid"`
	Recommendations []int32   `json:"recommendations"`
}
