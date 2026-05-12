package route

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "gitlab.com/hannlync/backend/stream-go.git"
)

func ackFieldPatternForEvent(id string) string {
	return id + ":*"
}

func deleteEventAckMetadata(ctx context.Context, rdb *redis.Client, stream, id string) error {
	ackKey := fmt.Sprintf("%s:acks", stream)
	pattern := ackFieldPatternForEvent(id)
	var cursor uint64

	for {
		fields, nextCursor, err := rdb.HScanNoValues(ctx, ackKey, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(fields) > 0 {
			if err := rdb.HDel(ctx, ackKey, fields...).Err(); err != nil {
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			return nil
		}
	}
}

func parseEventPayload(body io.Reader, contentType string) (map[string]any, error) {
	var rawPayload []byte

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(contentType, "multipart/form-data") {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}

		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			return nil, err
		}
		rawPayload = []byte(strings.TrimSpace(values.Get("payload")))
	} else {
		var err error
		rawPayload, err = io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		rawPayload = []byte(strings.TrimSpace(string(rawPayload)))
	}

	if len(rawPayload) == 0 {
		return nil, errors.New("payload is required")
	}

	var decoded map[string]any
	if err := json.Unmarshal(rawPayload, &decoded); err != nil {
		return nil, fmt.Errorf("payload must be a valid JSON object: %w", err)
	}

	if len(decoded) == 0 {
		return nil, errors.New("payload must contain at least one field")
	}

	values := make(map[string]any, len(decoded))
	for key, value := range decoded {
		switch v := value.(type) {
		case nil:
			values[key] = ""
		case string:
			values[key] = v
		case float64:
			values[key] = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), ".")
		case bool:
			values[key] = fmt.Sprintf("%t", v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to encode field %q: %w", key, err)
			}
			values[key] = string(b)
		}
	}

	return values, nil
}

func RegisterAPI(r *gin.Engine, rdb *redis.Client, producer *stream.Producer) {
	api := r.Group("/api/v1")

	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	streams := api.Group("/streams")

	streams.POST("", func(c *gin.Context) {
		name := c.PostForm("name")
		id, err := rdb.XAdd(c, &redis.XAddArgs{
			Stream: name,
			MaxLen: 1000,
			Values: map[string]any{"": ""},
		}).Result()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		if err := rdb.XDel(c, name, id).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		c.Status(http.StatusOK)
	})

	streams.DELETE("/:stream", func(c *gin.Context) {
		stream := c.Param("stream")

		if err := rdb.Del(c, stream).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := rdb.Del(c, fmt.Sprintf("%s:acks", stream)).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(200)
	})

	streams.DELETE("/:stream/consumers/:group/:name", func(c *gin.Context) {
		stream := c.Param("stream")
		group := c.Param("group")
		name := c.Param("name")

		if err := rdb.XGroupDelConsumer(c, stream, group, name).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(200)
	})

	streams.POST("/:stream/events", func(c *gin.Context) {
		stream := c.Param("stream")
		values, err := parseEventPayload(c.Request.Body, c.GetHeader("Content-Type"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := producer.PushTo(c, stream, values); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "created"})
	})

	streams.DELETE("/:stream/events/:id", func(c *gin.Context) {
		stream := c.Param("stream")
		id := c.Param("id")

		if err := rdb.XDel(c, stream, id).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := deleteEventAckMetadata(c, rdb, stream, id); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
	})
}
