# 🍿 movie-site-api

Welcome to **movie-site-api**! This is a charming API built with the [Gin](https://gin-gonic.com/) framework in Go, serving up a delightful collection of movies from a MongoDB cluster.

<img src="https://avatars.githubusercontent.com/u/7894478?v=4" alt="Gin_Logo" width=100/><img alt="Toy Story" src="https://stickershop.line-scdn.net/sticonshop/v1/product/5b337485031a671b9c23d56d/iPhone/main.png" width="100" /><img src="https://miro.medium.com/v2/resize:fit:1000/0*YISbBYJg5hkJGcQd.png" alt="Go Gopher" width=100/><img src="https://stickershop.line-scdn.net/sticonshop/v1/product/5c7629e6031a6757b98a21a8/iPhone/main.png?v=3" alt="Pooh Bear" width="100"/><img src="https://stickershop.line-scdn.net/sticonshop/v1/product/5dedd7af031a67c29d105026/iPhone/main.png?v=2" alt="Star Wars" width="100"/>

## 🎬 About

This project is a RESTful API that connects to a MongoDB database to fetch and serve movie data. Each movie comes with a variety of information including scores, genres, plot, cast, and more!

## 🚀 Features

- Fetch a list of movies with all their details.
- Smooth integration with MongoDB.
- Built using the lightweight and fast [Gin](https://gin-gonic.com/) framework.

## 🛠️ Getting Started

### Prerequisites

- Go (version 1.22.3)
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

### Example Endpoint

- **Get Movies**: Fetch a list of all movies

  GET /movies

  Response:

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
  "plot": "An example plot.",
  "poster": "http://example.com/poster.jpg",
  "actors": "John Doe, Jane Doe",
  "director": "Director Name",
  "ratings": "5 stars",
  "boxoffice": "$1,000,000",
  "rated": "PG-13",
  "runtime": 120,
  "provider": "Example Provider",
  "budget": "$100,000,000",
  "tmdbid": 123456,
  "recommendations": "['Another Movie']"
  }
