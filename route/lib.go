package route

import (
	"context"
	"encoding/json"
	"errors"
	"event-manager/utils"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
)

func ackFieldPatternForEvent(id string) string {
	return id + ":*"
}

func splitAckField(field string) (string, string, bool) {
	idEnd := strings.Index(field, ":")
	if idEnd == -1 {
		return "", "", false
	}
	eventID := field[:idEnd]
	group := field[idEnd+1:]
	if eventID == "" || group == "" || !strings.Contains(eventID, "-") {
		return "", "", false
	}
	return eventID, group, true
}

func buildAckMetadata(values map[string]string) map[string]AckMetadata {
	acks := make(map[string]AckMetadata)
	for field, raw := range values {
		eventID, group, ok := splitAckField(field)
		if !ok {
			continue
		}

		var ack struct {
			Group     string `json:"group"`
			Consumer  string `json:"consumer"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(raw), &ack); err != nil {
			continue
		}
		if ack.Group != "" {
			group = ack.Group
		}

		metadata := acks[eventID]
		metadata.EventID = eventID
		metadata.Records = append(metadata.Records, AckRecord{
			Group:     group,
			Consumer:  ack.Consumer,
			Timestamp: ack.Timestamp,
		})
		acks[eventID] = metadata
	}

	for eventID, metadata := range acks {
		sort.Slice(metadata.Records, func(i, j int) bool {
			if metadata.Records[i].Group != metadata.Records[j].Group {
				return metadata.Records[i].Group < metadata.Records[j].Group
			}
			return metadata.Records[i].Consumer < metadata.Records[j].Consumer
		})
		acks[eventID] = metadata
	}
	return acks
}

func buildAckStatus(ack AckMetadata, groups []string) StreamAckStatus {
	if len(ack.Records) == 0 {
		return StreamAckStatus{
			Label:   "Pending",
			State:   "pending",
			Tooltip: "Not processed yet\nNo consumer group has acked this event",
		}
	}

	ackedGroups := make(map[string]AckRecord, len(ack.Records))
	for _, record := range ack.Records {
		if record.Group != "" {
			ackedGroups[record.Group] = record
		}
	}

	totalGroups := len(groups)
	ackedCount := len(ackedGroups)
	if totalGroups == 0 {
		ackedCount = len(ack.Records)
	}

	label := fmt.Sprintf("Acked %d", ackedCount)
	if totalGroups > 0 {
		label = fmt.Sprintf("Acked %d/%d", ackedCount, totalGroups)
	}

	state := "partial"
	if totalGroups > 0 && ackedCount >= totalGroups {
		state = "complete"
	}

	var processed []string
	for _, record := range ack.Records {
		group := record.Group
		if group == "" {
			group = "Unknown group"
		}
		consumer := record.Consumer
		if consumer == "" {
			consumer = "Unknown consumer"
		}
		processed = append(processed, fmt.Sprintf("%s: %s", group, consumer))
		if record.Timestamp != "" {
			processed = append(processed, "Acked at "+record.Timestamp)
		}
	}

	tooltip := []string{"Processed by", strings.Join(processed, "\n")}
	if totalGroups > 0 {
		var pendingGroups []string
		for _, group := range groups {
			if _, ok := ackedGroups[group]; !ok {
				pendingGroups = append(pendingGroups, group)
			}
		}
		if len(pendingGroups) > 0 {
			tooltip = append(tooltip, "", "Still pending", strings.Join(pendingGroups, "\n"))
		}
	}

	return StreamAckStatus{
		Label:   label,
		State:   state,
		Tooltip: strings.Join(tooltip, "\n"),
	}
}

func groupNames(groups []redis.XInfoGroup) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	sort.Strings(names)
	return names
}

func buildStreamEventsResponse(streamName string, events []redis.XMessage, ackValues map[string]string, groups []string) StreamEventsResponse {
	headerSeen := make(map[string]bool)
	var primitiveHeaders []string
	var complexHeaders []string
	acks := buildAckMetadata(ackValues)
	rows := make([]StreamEventInfo, 0, len(events))

	for _, event := range events {
		row := StreamEventInfo{
			Timestamp: utils.TimeFromID(event.ID),
			ID:        event.ID,
			Values:    make(map[string]string, len(event.Values)),
			Ack:       buildAckStatus(acks[event.ID], groups),
		}

		for key, value := range event.Values {
			row.Values[key] = eventCellValue(value)
			if headerSeen[key] {
				continue
			}
			headerSeen[key] = true
			if isComplexEventValue(value) {
				complexHeaders = append(complexHeaders, key)
			} else {
				primitiveHeaders = append(primitiveHeaders, key)
			}
		}

		rows = append(rows, row)
	}

	sort.Strings(primitiveHeaders)
	sort.Strings(complexHeaders)
	headers := append(primitiveHeaders, complexHeaders...)

	return StreamEventsResponse{
		StreamName: streamName,
		Headers:    headers,
		Events:     rows,
	}
}

func eventCellValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return prettyJSONString(v)
	case []byte:
		return prettyJSONString(string(v))
	case fmt.Stringer:
		return v.String()
	default:
		if !isComplexEventValue(value) {
			return fmt.Sprintf("%v", value)
		}
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(b)
	}
}

func prettyJSONString(value string) string {
	trimmed := strings.TrimSpace(value)
	if !looksLikeJSONObject(trimmed) {
		return value
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return value
	}
	b, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return value
	}
	return string(b)
}

func isComplexEventValue(value any) bool {
	switch v := value.(type) {
	case string:
		return looksLikeJSONObject(strings.TrimSpace(v))
	case []byte:
		return looksLikeJSONObject(strings.TrimSpace(string(v)))
	case nil, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return false
	default:
		return true
	}
}

func looksLikeJSONObject(value string) bool {
	return strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[")
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
