# 🍿 movie-site-api

Welcome to **movie-site-api**! This is an API built with the [Gin](https://gin-gonic.com/) framework in Go, serving up a delightful collection of movies from a MongoDB cluster.

<img src="https://avatars.githubusercontent.com/u/7894478?v=4" alt="Gin_Logo" width=100/><img alt="Toy Story" src="https://stickershop.line-scdn.net/sticonshop/v1/product/5b337485031a671b9c23d56d/iPhone/main.png" width="100" /><img src="https://miro.medium.com/v2/resize:fit:1000/0*YISbBYJg5hkJGcQd.png" alt="Go Gopher" width=100/><img src="https://stickershop.line-scdn.net/sticonshop/v1/product/5c7629e6031a6757b98a21a8/iPhone/main.png?v=3" alt="Pooh Bear" width="100"/><img src="https://stickershop.line-scdn.net/sticonshop/v1/product/5dedd7af031a67c29d105026/iPhone/main.png?v=2" alt="Star Wars" width="100"/>

## 🎬 About

This project is a RESTful API that connects to a MongoDB database to fetch and serve movie data. Each movie comes with a variety of information including scores, genres, plot, cast, and more!

## 🚀 Features

- Fetch a list of movies with all their details.
- Smooth integration with MongoDB.
- Built using the lightweight and fast [Gin](https://gin-gonic.com/) framework.

## 🛠️ Getting Started

### Prerequisites

- Go (version 1.26 or newer)
- MongoDB instance
- Gin framework

### Installation

1. **Clone the repository:**

   ```sh
    git clone https://github.com/helfy18/movie-site-api.git
    cd movie-site-api
   ```

2. **Install dependencies:**

   ```sh
   go mod tidy
   ```

3. **Set up your MongoDB connection:**

   Ensure you have a MongoDB instance running and update your connection string in the application configuration.
   Contact me if you can be trusted with mine.

### Running the API

To start the API server, simply run:

    go run main.go

The server will start, and you'll be able to access the API at http://localhost:8080.

### Endpoints

| Method | Path                | Description                                                        |
| ------ | ------------------- | ------------------------------------------------------------------ |
| GET    | `/movies/list`      | List movies (filterable by genre, universe, year, runtime, etc.)   |
| GET    | `/movies/get`       | Get one movie by `tmdbid`, or by `title` and `year`                |
| GET    | `/movies/list/id`   | Get movies for one or more `tmdbid` values                         |
| GET    | `/movies/random`    | Get a random movie matching the same filters as `/movies/list`     |
| GET    | `/movies/mostRecent`| Get the most recently added movies (`count`, default 20)           |
| GET    | `/movies/count`     | Get the total number of movies                                     |
| GET    | `/types/list`       | Get distinct universes, genres, years, providers, studios, etc.    |
| POST   | `/auth/login`       | Log in                                                             |

### Example Response

Every movie endpoint returns the full movie object. For example, `GET /movies/get?tmdbid=123456`:

```json
{
  "movie": "Example Movie",
  "jh_score": 85,
  "universe": "Example Universe",
  "sub_universe": "Example Sub Universe",
  "genre": "Action",
  "genre_2": "Adventure",
  "holiday": "None",
  "exclusive": "No",
  "studio": "Example Studio",
  "year": 2024,
  "review": "Great movie!",
  "ranking": 42,
  "dani_approved": true,
  "plot": "An example plot.",
  "poster": "http://example.com/poster.jpg",
  "actors": "John Doe, Jane Doe",
  "director": "Director Name",
  "ratings": [
    { "source": "Internet Movie Database", "value": "8.5/10" },
    { "source": "Rotten Tomatoes", "value": "92%" },
    { "source": "Metacritic", "value": "80/100" }
  ],
  "boxoffice": "$1,000,000",
  "rated": "PG-13",
  "runtime": 120,
  "provider": {
    "link": "https://www.themoviedb.org/movie/123456/watch",
    "rent": [
      { "logo_path": "/example.jpg", "provider_id": 2, "provider_name": "Apple TV", "display_priority": 4 }
    ],
    "flatrate": [
      { "logo_path": "/example.jpg", "provider_id": 8, "provider_name": "Netflix", "display_priority": 0 }
    ],
    "buy": [
      { "logo_path": "/example.jpg", "provider_id": 10, "provider_name": "Amazon Video", "display_priority": 3 }
    ],
    "ads": [
      { "logo_path": "/example.jpg", "provider_id": 538, "provider_name": "Plex", "display_priority": 20 }
    ],
    "free": [
      { "logo_path": "/example.jpg", "provider_id": 300, "provider_name": "Pluto TV", "display_priority": 25 }
    ]
  },
  "budget": "$100,000,000",
  "tmdbid": 123456,
  "recommendations": [234567, 345678],
  "rottentomatoes": "92%",
  "imdb": "8.5",
  "metacritic": "80",
  "trailer": "https://www.youtube.com/watch?v=example",
  "origin_country": "US",
  "ms_added": 1700000000000
}
```

## Updating the movie data

The Google Sheet is the source of truth. `scripts/syncMovies.py` reads it, fills in
TMDB/OMDB details, writes them back to the sheet, and upserts everything into Mongo.

```shell
pip3 install -r scripts/requirements.txt   # once
python3 scripts/syncMovies.py --check      # read-only connection test
python3 scripts/syncMovies.py --new-only   # after adding movies to the sheet
python3 scripts/syncMovies.py --skip-enrich  # after reordering or editing scores/reviews
python3 scripts/syncMovies.py              # full refresh (OMDB only for recent movies)
python3 scripts/syncMovies.py --backup     # snapshot Mongo + sheet to scripts/backups/
```

Settings live in `.env` next to this README (gitignored); the header of the script lists the keys.
The `CAST` table in the script must match the `movie` struct in `modules/movies/types.go`.
