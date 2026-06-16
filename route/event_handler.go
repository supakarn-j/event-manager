package route

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/redis/go-redis/v9"
)

type EventHandler struct {
	rdb  *redis.Client
	node *centrifuge.Node
}

func newEventHandler(rdb *redis.Client, node *centrifuge.Node) EventHandler {
	return EventHandler{
		rdb:  rdb,
		node: node,
	}
}

func (h *EventHandler) handleConsumerHealthyEvent(msg string) {
	var consumerStatus ConsumerStatus
	if err := json.Unmarshal([]byte(msg), &consumerStatus); err != nil {
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}
	log.Printf("status: %+v", consumerStatus)

	if err := h.setConsumerStatus(consumerStatus); err != nil {
		log.Printf("Failed to set consumer status: %+v", err)
		return
	}

	payload := healthyConsumerStatusPayload(consumerStatus)
	h.sendOverWs(payload, consumerStatus.Streams...)
}

func (h *EventHandler) handleConsumerAckReport(msg string) {
	var ackReport AckReport
	if err := json.Unmarshal([]byte(msg), &ackReport); err != nil {
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}

	if err := setAckReport(h.rdb, ackReport); err != nil {
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}
}

func (h *EventHandler) handleNewEventAdded(msg string) {
	stream := msg
	res, err := h.rdb.XRevRangeN(context.Background(), stream, "+", "-", 1).Result()
	if err != nil {
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}
	if len(res) == 0 {
		err := fmt.Errorf("no messages found in stream %s", stream)
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}

	log.Printf("msg: %+v", res)
	event := res[0]
	payload := newEventAddedPayload(event)
	h.sendOverWs(payload, stream)
}

func (h *EventHandler) handleConsumerExpired(msg string) {
	streamKey := msg + ":streams"
	streams, err := h.rdb.SMembers(context.Background(), streamKey).Result()
	if err != nil {
		log.Printf("Failed to unmarshal message: %+v", err)
		return
	}

	payload := expiredConsumerPayload(msg)
	h.sendOverWs(payload, streams...)

}

func (h *EventHandler) sendOverWs(payload any, channels ...string) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal consumer status payload: %v", err)
		return
	}

	for _, ch := range channels {
		_, err := h.node.Publish(ch, payloadJSON)
		if err != nil {
			log.Printf("Failed to publish consumer status to %s: %v", ch, err)
			continue
		}
		log.Printf("Published consumer status for %s to stream %s", channels, ch)
	}
}

func (h *EventHandler) setConsumerStatus(consumer ConsumerStatus) error {
	key := fmt.Sprintf("consumer:health:%s:%s", consumer.Group, consumer.ConsumerName)
	if err := h.rdb.HSet(context.Background(), key,
		map[string]any{
			"healthy":  consumer.Status == "up",
			"lastSeen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
			"ip":       consumer.IP,
		},
	).Err(); err != nil {
		log.Printf("Failed to set consumer health status in Redis: %v", err)
		return err
	}

	streamKey := key + ":streams"
	if err := h.rdb.SAdd(context.Background(), streamKey, consumer.Streams).Err(); err != nil {
		return err
	}

	h.rdb.HExpire(context.Background(), key, 30*time.Second, "lastSeen")

	return nil
}

func setAckReport(rdb *redis.Client, ackReport AckReport) error {
	field := fmt.Sprintf("%s:%s", ackReport.MsgID, ackReport.Group)
	value := map[string]any{
		"group":     ackReport.Group,
		"consumer":  ackReport.ConsumerName,
		"timestamp": ackReport.TimeStamp,
	}

	valueStr, _ := json.Marshal(value)
	ackReportKey := fmt.Sprintf("consumer:ack:%s", ackReport.Stream)
	return rdb.HSet(context.Background(), ackReportKey, field, valueStr).Err()
}
