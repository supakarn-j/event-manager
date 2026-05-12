package route

import (
	"encoding/json"
	"errors"
	"event-manager/utils"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type StreamListItem struct {
	Name        string
	DisplayName string
	Length      int64
	Groups      int64
}

type StreamTreeNode struct {
	Name        string
	DisplayName string
	FullName    string
	Length      int64
	Groups      int64
	IsStream    bool
	Children    []StreamTreeNode
}

type StreamTree struct {
	Nodes        []StreamTreeNode
	TotalStreams int
}

type EventTable struct {
	Headers []string
	Rows    []EventTableRow
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

type EventTableRow struct {
	Timestamp  string
	ID         string
	Values     map[string]string
	AckLabel   string
	AckState   string
	AckTooltip string
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

func groupNames(groups []redis.XInfoGroup) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	sort.Strings(names)
	return names
}

func buildAckStatus(ack AckMetadata, groups []string) (string, string, string) {
	if len(ack.Records) == 0 {
		return "Pending", "pending", "Not processed yet\nNo consumer group has acked this event"
	}

	ackedGroups := make(map[string]AckRecord, len(ack.Records))
	for _, record := range ack.Records {
		if record.Group == "" {
			continue
		}
		ackedGroups[record.Group] = record
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
		if record.Consumer != "" {
			consumer = record.Consumer
		}
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

	return label, state, strings.Join(tooltip, "\n")
}

func buildEventTable(events []redis.XMessage, acks map[string]AckMetadata, groups []string) EventTable {
	headerSeen := make(map[string]bool)
	var primitiveHeaders []string
	var complexHeaders []string
	rows := make([]EventTableRow, 0, len(events))

	for _, event := range events {
		row := EventTableRow{
			Timestamp: utils.TimeFromID(event.ID),
			ID:        event.ID,
			Values:    make(map[string]string, len(event.Values)),
			AckLabel:  "Pending",
			AckState:  "pending",
		}
		if ack, ok := acks[event.ID]; ok {
			row.AckLabel, row.AckState, row.AckTooltip = buildAckStatus(ack, groups)
		} else {
			row.AckTooltip = "Not processed yet\nNo consumer group has acked this event"
		}

		for key, value := range event.Values {
			row.Values[key] = eventCellValue(value)
			if headerSeen[key] {
				continue
			}
			headerSeen[key] = true
			if isPrimitiveEventValue(value) {
				primitiveHeaders = append(primitiveHeaders, key)
			} else {
				complexHeaders = append(complexHeaders, key)
			}
		}

		rows = append(rows, row)
	}

	sort.Strings(primitiveHeaders)
	sort.Strings(complexHeaders)
	headers := append(primitiveHeaders, complexHeaders...)

	return EventTable{
		Headers: headers,
		Rows:    rows,
	}
}

func eventCellValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		if isPrimitiveEventValue(value) {
			return fmt.Sprintf("%v", value)
		}
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(b)
	}
}

func isPrimitiveEventValue(value any) bool {
	switch value.(type) {
	case nil, string, []byte, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func buildStreamTree(streams []StreamListItem) StreamTree {
	var nodes []StreamTreeNode
	for _, stream := range streams {
		nodes = insertStreamNode(nodes, stream)
	}
	sortStreamNodes(nodes)

	return StreamTree{
		Nodes:        nodes,
		TotalStreams: len(streams),
	}
}

func insertStreamNode(nodes []StreamTreeNode, stream StreamListItem) []StreamTreeNode {
	parts := strings.Split(stream.Name, ":")
	for _, part := range parts {
		if part == "" {
			return append(nodes, streamLeafNode(stream, stream.Name))
		}
	}

	if len(parts) == 1 {
		return append(nodes, streamLeafNode(stream, stream.Name))
	}

	groupIndex := findStreamNode(nodes, parts[0])
	if groupIndex == -1 {
		nodes = append(nodes, StreamTreeNode{
			Name:        parts[0],
			DisplayName: parts[0],
		})
		groupIndex = len(nodes) - 1
	}

	nodes[groupIndex].Children = insertStreamNodePart(nodes[groupIndex].Children, parts[1:], stream)
	return nodes
}

func insertStreamNodePart(nodes []StreamTreeNode, parts []string, stream StreamListItem) []StreamTreeNode {
	if len(parts) == 1 {
		return append(nodes, streamLeafNode(stream, parts[0]))
	}

	groupIndex := findStreamNode(nodes, parts[0])
	if groupIndex == -1 {
		nodes = append(nodes, StreamTreeNode{
			Name:        parts[0],
			DisplayName: parts[0],
		})
		groupIndex = len(nodes) - 1
	}

	nodes[groupIndex].Children = insertStreamNodePart(nodes[groupIndex].Children, parts[1:], stream)
	return nodes
}

func streamLeafNode(stream StreamListItem, displayName string) StreamTreeNode {
	return StreamTreeNode{
		Name:        displayName,
		DisplayName: displayName,
		FullName:    stream.Name,
		Length:      stream.Length,
		Groups:      stream.Groups,
		IsStream:    true,
	}
}

func findStreamNode(nodes []StreamTreeNode, name string) int {
	for i, node := range nodes {
		if node.Name == name && !node.IsStream {
			return i
		}
	}
	return -1
}

func sortStreamNodes(nodes []StreamTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsStream != nodes[j].IsStream {
			return !nodes[i].IsStream
		}
		return nodes[i].DisplayName < nodes[j].DisplayName
	})
	for i := range nodes {
		sortStreamNodes(nodes[i].Children)
	}
}

func createMyRenderer() multitemplate.Renderer {
	renderer := multitemplate.NewRenderer()
	renderer.AddFromFiles("home", "templates/base.html", "templates/nav.html", "templates/home.html")
	renderer.AddFromFiles("streams", "templates/streams/index.html")
	renderer.AddFromFiles("streams_name", "templates/base.html", "templates/nav.html", "templates/streams/name.html")
	funcMap := template.FuncMap{
		"streamIDTime": utils.TimeFromID,
		"prettyJSON": func(v any) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return "{}"
			}
			return string(b)
		},
	}

	tmpl := template.Must(template.New("name_events.html").Funcs(funcMap).ParseFiles("templates/streams/name_events.html"))
	renderer.Add("streams_name_events", tmpl)

	return renderer
}

func RegisterFrontend(r *gin.Engine, rdb *redis.Client) {
	r.HTMLRender = createMyRenderer()
	r.Static("/static", "static")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/home")
	})

	r.GET("/home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home", map[string]any{
			"title":         "Home",
			"showAddStream": true,
		})
	})

	r.POST("/streams", func(c *gin.Context) {
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
	})

	r.GET("/streams", func(c *gin.Context) {
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
		tree := buildStreamTree(streams)
		c.HTML(http.StatusOK, "streams", map[string]any{
			"tree":     tree,
			"redisURL": "redis://" + rdb.Options().Addr,
		})
	})

	type Consumer struct {
		Name     string
		LastSeen string
		Healthy  bool
		Pending  int64
	}

	type Group struct {
		Name      string
		Pending   int64
		Consumers []Consumer
	}

	r.GET("/streams/:name", func(c *gin.Context) {
		streamName := c.Param("name")

		stream, err := rdb.XInfoStreamFull(c, streamName, 0).Result()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		var groups []Group
		for _, g := range stream.Groups {
			var pendingCount int64
			pending, err := rdb.XPending(c, streamName, g.Name).Result()
			if err == nil && pending != nil {
				pendingCount = pending.Count
			}

			var consumers []Consumer
			for _, cmr := range g.Consumers {
				var healthy bool
				var consumerPending int64
				if pending != nil {
					consumerPending = pending.Consumers[cmr.Name]
				}
				res, _ := rdb.HGet(c, fmt.Sprintf("%s:%s:%s", streamName, g.Name, cmr.Name), "timestamp").Result()
				lastSeen := cmr.SeenTime.Format("2006-01-02 15:04:05 -07:00 MST")
				if res != "" {
					healthy = true
					unix, _ := strconv.ParseInt(res, 10, 64)
					lastSeen = time.Unix(int64(unix), 0).Format("2006-01-02 15:04:05 -07:00 MST")
				}

				consumers = append(consumers, Consumer{
					Name:     cmr.Name,
					LastSeen: lastSeen,
					Healthy:  healthy,
					Pending:  consumerPending,
				})
			}
			groups = append(groups, Group{
				Name:      g.Name,
				Pending:   pendingCount,
				Consumers: consumers,
			})
		}
		c.HTML(http.StatusOK, "streams_name", map[string]any{
			"title":  streamName,
			"name":   streamName,
			"groups": groups,
		})
	})

	r.GET("/streams/:name/events", func(c *gin.Context) {
		streamName := c.Param("name")
		if streamName == "" {
			c.AbortWithError(http.StatusBadRequest, errors.New("streamName is required"))
		}

		res, err := rdb.XRevRange(c, streamName, "+", "-").Result()

		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ackValues, _ := rdb.HGetAll(c, fmt.Sprintf("%s:acks", streamName)).Result()
		acks := buildAckMetadata(ackValues)
		streamGroups, _ := rdb.XInfoGroups(c, streamName).Result()
		groups := groupNames(streamGroups)

		c.HTML(http.StatusOK, "streams_name_events", struct {
			StreamName string
			Events     []redis.XMessage
			Table      EventTable
		}{
			StreamName: streamName,
			Events:     res,
			Table:      buildEventTable(res, acks, groups),
		})
	})

	r.DELETE("/streams/:name", func(c *gin.Context) {
		name := c.Param("name")

		if err := rdb.Del(c, name).Err(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		c.Status(200)
	})
}
