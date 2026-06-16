package route

import (
	"event-manager/service"
	"event-manager/service/client"

	"github.com/gin-gonic/gin"
)

func registerApiRoute(r *Route) {
	api := r.router.Group("/api/v1")

	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	c := client.NewRedisClient(r.rdb)
	svc := service.NewAPIService(c)

	handler := newApiHandler(svc, r.producer)

	streams := api.Group("/streams")
	// Create Stream
	streams.POST("", handler.createStreamHandler)
	// List Streams
	streams.GET("", handler.listStreamsHandler)
	// Get Stream Info
	streams.GET("/:stream", handler.getStreamInfoHandler)
	// Delete Stream
	streams.DELETE("/:stream", handler.deleteStreamHandler)
	// Publish Event
	streams.POST("/:stream/events", handler.pusblishEventHandler)
	// List Events
	streams.GET("/:stream/events", handler.listEvents)
	// Delete Event
	streams.DELETE("/:stream/events/:id", handler.deleteEvent)
	// Delete Consumer
	streams.DELETE("/:stream/consumers/:group/:name", handler.deleteConsumer)
}
