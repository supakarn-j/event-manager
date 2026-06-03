package route

import (
	"event-manager/utils"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "github.com/supakarn-j/stream-go"
)

type StreamDetailResponse struct {
	Name   string                `json:"name"`
	Groups []StreamConsumerGroup `json:"groups"`
}

type StreamConsumerGroup struct {
	Name      string           `json:"name"`
	Pending   int64            `json:"pending"`
	Consumers []StreamConsumer `json:"consumers"`
}

type StreamConsumer struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	LastSeen string `json:"lastSeen"`
	Healthy  bool   `json:"healthy"`
	Pending  int64  `json:"pending"`
}

type StreamEventsResponse struct {
	StreamName string            `json:"streamName"`
	Headers    []string          `json:"headers"`
	Events     []StreamEventInfo `json:"events"`
}

type StreamEventInfo struct {
	Timestamp string            `json:"timestamp"`
	ID        string            `json:"id"`
	Values    map[string]string `json:"values"`
	Ack       StreamAckStatus   `json:"ack"`
}

type StreamAckStatus struct {
	Label   string `json:"label"`
	State   string `json:"state"`
	Tooltip string `json:"tooltip"`
}

type AckMetadata struct {
	EventID string
	Records []AckRecord
}

type AckRecord struct {
	Group     string
	Consumer  string
	Timestamp string
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

	streams.GET("", func(c *gin.Context) {
		var curr uint64
		var streams []StreamListItem

		for {
			keys, nextCurr, err := rdb.Scan(c, curr, "*", 0).Result()
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			}

			for _, key := range keys {
				t, err := rdb.Type(c, key).Result()
				if err != nil {
					continue
				}

				if t == "stream" {
					s, err := rdb.XInfoStream(c, key).Result()
					if err != nil {
						continue
					}
					streams = append(streams, StreamListItem{
						Name:   key,
						Length: s.Length,
						Groups: s.Groups,
					})
				}
			}
			curr = nextCurr
			if curr == 0 {
				break
			}
		}
		c.JSON(http.StatusOK, StreamsResponse{
			RedisURL: "redis://" + rdb.Options().Addr,
			Streams:  streams,
		})
	})

	streams.GET("/:stream", func(c *gin.Context) {
		streamName := c.Param("stream")

		stream, err := rdb.XInfoStreamFull(c, streamName, 0).Result()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		groups := make([]StreamConsumerGroup, 0, len(stream.Groups))
		for _, group := range stream.Groups {
			var pendingCount int64
			pending, err := rdb.XPending(c, streamName, group.Name).Result()
			if err == nil && pending != nil {
				pendingCount = pending.Count
			}

			consumers := make([]StreamConsumer, 0, len(group.Consumers))
			for _, consumer := range group.Consumers {
				var healthy bool
				var consumerPending int64
				if pending != nil {
					consumerPending = pending.Consumers[consumer.Name]
				}

				lastSeen := consumer.SeenTime.Format("2006-01-02 15:04:05 -07:00 MST")
				b64StreamName := utils.Base64Encode(streamName)
				key := fmt.Sprintf("consumer:health:%s:%s:%s", b64StreamName, group.Name, consumer.Name)
				res, _ := rdb.HGet(c, key, "lastSeen").Result()
				if res != "" {
					healthy = true
					lastSeen = res
				}
				log.Printf("res: %s", res)

				ip, _ := rdb.HGet(c, key, "ip").Result()

				consumers = append(consumers, StreamConsumer{
					Name:     consumer.Name,
					IP:       ip,
					LastSeen: lastSeen,
					Healthy:  healthy,
					Pending:  consumerPending,
				})
			}

			groups = append(groups, StreamConsumerGroup{
				Name:      group.Name,
				Pending:   pendingCount,
				Consumers: consumers,
			})
		}

		c.JSON(http.StatusOK, StreamDetailResponse{
			Name:   streamName,
			Groups: groups,
		})
	})

	streams.GET("/:stream/events", func(c *gin.Context) {
		streamName := c.Param("stream")
		if streamName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stream is required"})
			return
		}

		events, err := rdb.XRevRange(c, streamName, "+", "-").Result()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ackValues, _ := rdb.HGetAll(c, fmt.Sprintf("acks:%s", streamName)).Result()
		streamGroups, _ := rdb.XInfoGroups(c, streamName).Result()

		c.JSON(http.StatusOK, buildStreamEventsResponse(streamName, events, ackValues, groupNames(streamGroups)))
	})

	streams.DELETE("/:stream", func(c *gin.Context) {
		stream := c.Param("stream")

		if err := rdb.Del(c, stream).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := rdb.Del(c, fmt.Sprintf("acks:%s", stream)).Err(); err != nil {
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
