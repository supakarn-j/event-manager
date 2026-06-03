package main

import (
	"context"
	"encoding/json"
	"event-manager/utils"
	"fmt"
	"log"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/redis/go-redis/v9"
)

type ConsumerStatus struct {
	ConsumerName string   `json:"name"`
	Status       string   `json:"status"`
	Group        string   `json:"group"`
	IP           string   `json:"ip"`
	Streams      []string `json:"streams"`
}

type eventHandler func(rdb *redis.Client, msg string) (map[string]any, []string, error)

var eventHandlers = map[string]eventHandler{
	"consumer:status":         buildConsumerStatusPayload,
	"__keyevent@0__:xadd":     buildEventAddPayload,
	"__keyevent@0__:hexpired": buildHashExpiredPayload,
}

func setConsumerStatus(rdb *redis.Client, consumer ConsumerStatus) error {
	for _, stream := range consumer.Streams {
		base64Stream := utils.Base64Encode(stream)
		key := fmt.Sprintf("consumer:health:%s:%s:%s", base64Stream, consumer.Group, consumer.ConsumerName)
		if err := rdb.HSet(context.Background(), key,
			map[string]any{
				"healthy":  consumer.Status == "up",
				"lastSeen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
				"ip":       consumer.IP,
			},
		).Err(); err != nil {
			log.Printf("Failed to set consumer health status in Redis: %v", err)
			return err
		}

		rdb.HExpire(context.Background(), key, 30*time.Second, "lastSeen")
	}

	return nil
}

func buildConsumerStatusPayload(rdb *redis.Client, msg string) (map[string]any, []string, error) {
	var consumerStatus ConsumerStatus
	if err := json.Unmarshal([]byte(msg), &consumerStatus); err != nil {
		return nil, nil, err
	}

	if err := setConsumerStatus(rdb, consumerStatus); err != nil {
		return nil, nil, err
	}

	payload := map[string]any{
		"action":    "health_check",
		"group":     consumerStatus.Group,
		"consumer":  consumerStatus.ConsumerName,
		"ip":        consumerStatus.IP,
		"healthy":   consumerStatus.Status == "up",
		"last_seen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
	}

	return payload, consumerStatus.Streams, nil
}

func buildHashExpiredPayload(rdb *redis.Client, msg string) (map[string]any, []string, error) {
	b64Stream, group, consumerName := utils.ParseExpiredKey(msg)
	stream, _ := utils.Base64Decode(b64Stream)
	payload := map[string]any{
		"action":    "health_check",
		"group":     group,
		"consumer":  consumerName,
		"healthy":   false,
		"last_seen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
	}

	return payload, []string{stream}, nil
}

func buildEventAddPayload(rdb *redis.Client, msg string) (map[string]any, []string, error) {
	stream := msg
	res, err := rdb.XRevRangeN(context.Background(), stream, "+", "-", 1).Result()
	if err != nil {
		return nil, nil, err
	}
	if len(res) == 0 {
		return nil, nil, fmt.Errorf("no messages found in stream %s", stream)
	}

	log.Printf("msg: %+v", res)
	event := res[0]
	payload := map[string]any{
		"action":    "event_added",
		"timestamp": utils.TimeFromID(event.ID),
		"id":        event.ID,
		"values":    event.Values,
	}

	return payload, []string{stream}, nil
}

func listenEvents(ctx context.Context, rdb *redis.Client, node *centrifuge.Node) {
	listeningCh := utils.ExtractMapKeys(eventHandlers)
	sub := rdb.PSubscribe(ctx, listeningCh...)
	ch := sub.Channel()

	for msg := range ch {
		if _, ok := eventHandlers[msg.Channel]; !ok {
			log.Printf("No handler for channel %s, skipping", msg.Channel)
			continue
		}

		payload, streams, err := eventHandlers[msg.Channel](rdb, msg.Payload)
		if err != nil {
			log.Printf("Failed to process event for channel %s: %v", msg.Channel, err)
			continue
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Failed to marshal consumer status payload: %v", err)
			continue
		}

		for _, stream := range streams {
			_, err := node.Publish(stream, payloadJSON)
			if err != nil {
				log.Printf("Failed to publish consumer status to %s: %v", stream, err)
				continue
			}
			// log.Printf("Published consumer status for %s to stream %s", streams, stream)

		}
	}
}
