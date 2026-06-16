package client

type StreamInfo struct {
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

type StreamEventInfo struct {
	ID        string              `json:"id"`
	Values    map[string]any      `json:"values"`
	Consumers []ConsumerAckReport `json:"consumers"`
}

type ConsumerAckReport struct {
	Consumer  string
	Group     string
	Timestamp string
}

type StreamListItem struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Length      int64  `json:"length"`
	Groups      int64  `json:"groups"`
}
