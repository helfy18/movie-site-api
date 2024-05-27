package main

import (
	"github.com/gin-gonic/gin"
	"github.com/helfy18/movie-site-api/modules/movies"
)

func main() {
	router := gin.Default()
	router.GET("/albums", movies.GetAlbums)

	router.Run()
}
