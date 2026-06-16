package route

import (
	"event-manager/service/client"

	"github.com/redis/go-redis/v9"
)

type StreamsResponse struct {
	RedisURL string                  `json:"redisUrl"`
	Streams  []client.StreamListItem `json:"streams"`
}

type ConsumerStatus struct {
	ConsumerName string   `json:"name"`
	Status       string   `json:"status"`
	Group        string   `json:"group"`
	IP           string   `json:"ip"`
	Streams      []string `json:"streams"`
}

type AckReport struct {
	ConsumerName string `json:"consumer"`
	Group        string `json:"group"`
	MsgID        string `json:"msgId"`
	Stream       string `json:"stream"`
	TimeStamp    string `json:"timestamp"`
}

type eventHandlerFunc func(rdb *redis.Client, msg string) (map[string]any, []string, error)

type eventMap map[string]eventHandlerFunc
