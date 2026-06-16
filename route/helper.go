package route

import (
	"encoding/json"
	"errors"
	"event-manager/utils"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

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

func healthyConsumerStatusPayload(consumerStatus ConsumerStatus) map[string]any {
	return map[string]any{
		"action":    "health_check",
		"group":     consumerStatus.Group,
		"consumer":  consumerStatus.ConsumerName,
		"ip":        consumerStatus.IP,
		"healthy":   consumerStatus.Status == "up",
		"last_seen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
	}
}

func newEventAddedPayload(e redis.XMessage) map[string]any {
	return map[string]any{
		"action":    "event_added",
		"timestamp": utils.TimeFromID(e.ID),
		"id":        e.ID,
		"values":    e.Values,
	}
}

func expiredConsumerPayload(msg string) map[string]any {
	return map[string]any{
		"action":    "health_check",
		"group":     utils.GetGroupFromPubSub(msg),
		"consumer":  utils.GetConsumerFromPubSub(msg),
		"healthy":   false,
		"last_seen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
	}
}
