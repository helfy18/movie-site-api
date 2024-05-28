# 🍿 movie-site-api

Welcome to **movie-site-api**! This is a charming little API built with the [Gin](https://gin-gonic.com/) framework in Go, serving up a delightful collection of movies from a MongoDB cluster.

![Gin Logo](https://gin-gonic.com/images/logo.jpg)

## 🎬 About

This project is a RESTful API that connects to a MongoDB database to fetch and serve movie data. Each movie comes with a variety of information including scores, genres, plot, cast, and more!

## 🚀 Features

- Fetch a list of movies with all their details.
- Smooth integration with MongoDB.
- Built using the lightweight and fast [Gin](https://gin-gonic.com/) framework.

## 🛠️ Getting Started

### Prerequisites

- Go (version 1.16+)
- MongoDB instance
- Gin framework

### Installation

1. **Clone the repository:**

    git clone https://github.com/helfy18/movie-site-api.git
    cd movie-site-api

2. **Install dependencies:**

    go mod tidy

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
        "jv_score": 90,
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
