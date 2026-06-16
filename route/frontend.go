package route

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func spaHandler(distFS fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(distFS))

	return func(c *gin.Context) {
		if _, err := distFS.Open(c.Request.URL.Path[1:]); err != nil {
			c.Request.URL.Path = "/"
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}

func registerFrontend(r *gin.Engine, rdb *redis.Client, distFS fs.FS) {
	r.Static("/static", "static")
	r.Static("/assets", "frontend/dist/assets")
	r.StaticFile("/favicon.svg", "frontend/dist/favicon.svg")
	r.StaticFile("/icons.svg", "frontend/dist/icons.svg")

	renderApp := spaHandler(distFS)
	r.GET("/", renderApp)
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		renderApp(c)
	})
}
