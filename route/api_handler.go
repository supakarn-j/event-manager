package route

import (
	"event-manager/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/supakarn-j/stream-go"
)

type ApiHandler struct {
	service  service.APIService
	producer *stream.Producer
}

func newApiHandler(service service.APIService, producer *stream.Producer) *ApiHandler {
	return &ApiHandler{
		service:  service,
		producer: producer,
	}
}

func (h *ApiHandler) createStreamHandler(c *gin.Context) {
	name := c.PostForm("name")

	if err := h.service.CreateNewStream(c, name); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.Status(http.StatusCreated)
}

func (h *ApiHandler) listStreamsHandler(c *gin.Context) {
	streams, connStr, err := h.service.ListStreams(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, StreamsResponse{
		RedisURL: connStr,
		Streams:  streams,
	})
}

func (h *ApiHandler) getStreamInfoHandler(c *gin.Context) {
	streamName := c.Param("stream")

	streamInfo, err := h.service.GetStreamInfo(c, streamName)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, streamInfo)
}

func (h *ApiHandler) deleteStreamHandler(c *gin.Context) {
	stream := c.Param("stream")

	if err := h.service.DeleteStream(c, stream); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(200)
}

func (h *ApiHandler) listEvents(c *gin.Context) {
	streamName := c.Param("stream")
	if streamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stream is required"})
		return
	}

	events, err := h.service.ListEvents(c, streamName)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, events)
}

func (h *ApiHandler) pusblishEventHandler(c *gin.Context) {
	stream := c.Param("stream")
	values, err := parseEventPayload(c.Request.Body, c.GetHeader("Content-Type"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.producer.PushTo(c, stream, values); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "created"})

}

func (h *ApiHandler) deleteEvent(c *gin.Context) {
	stream := c.Param("stream")
	id := c.Param("id")

	if err := h.service.DeleteEvent(c, stream, id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *ApiHandler) deleteConsumer(c *gin.Context) {
	stream := c.Param("stream")
	group := c.Param("group")
	name := c.Param("name")

	if err := h.service.DeleteConsumer(c, stream, group, name); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Status(200)
}
