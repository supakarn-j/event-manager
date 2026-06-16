package client

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	ackReportKeyTmpl string = "consumer:ack:%s"
	consumerKeyTmpl  string = "consumer:health:%s:%s"
	streamsKeyTmpl   string = "consumer:health:%s:%s:streams"
)

type Client interface {
	GetRedisConnectionString() string
	CreateNewStream(ctx context.Context, name string, maxLen int64) (string, error)
	ListAllStreams(ctx context.Context) ([]StreamListItem, error)
	GetFullStreamInfo(ctx context.Context, streamName string) (StreamInfo, error)
	WipeAckMeta(ctx context.Context, streamName string) error
	DeleteKeyWithPattern(ctx context.Context, pattern string) error
	DeleteEvent(ctx context.Context, stream string, id ...string) error
	GetAllEvents(ctx context.Context, streamName string) ([]StreamEventInfo, error)
	DeleteAckMeta(ctx context.Context, streamName, id string) error
	DeleteConsumer(ctx context.Context, streamName, group, consumer string) error
}

type RedisClient struct {
	rdb *redis.Client
}

func (c *RedisClient) GetRedisConnectionString() string {
	return "redis://" + c.rdb.Options().Addr
}

func NewRedisClient(rdb *redis.Client) *RedisClient {
	return &RedisClient{
		rdb: rdb,
	}
}

func (c *RedisClient) CreateNewStream(ctx context.Context, stream string, maxLen int64) (string, error) {
	id, err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 1000,
		Values: map[string]any{"": ""},
	}).Result()
	if err != nil {
		return "", err
	}

	return id, nil
}

func (c *RedisClient) ListAllStreams(ctx context.Context) ([]StreamListItem, error) {
	var curr uint64
	var streams []StreamListItem

	for {
		keys, nextCurr, err := c.rdb.ScanType(ctx, curr, "*", 0, "stream").Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			s, err := c.rdb.XInfoStream(ctx, key).Result()
			if err != nil {
				continue
			}
			streams = append(streams, StreamListItem{
				Name:   key,
				Length: s.Length,
				Groups: s.Groups,
			})
		}
		curr = nextCurr
		if curr == 0 {
			break
		}
	}

	return streams, nil
}

func (c *RedisClient) DeleteEvent(ctx context.Context, stream string, id ...string) error {
	return c.rdb.XDel(ctx, stream, id...).Err()
}

func (c *RedisClient) GetFullStreamInfo(ctx context.Context, streamName string) (StreamInfo, error) {
	stream, err := c.rdb.XInfoStreamFull(ctx, streamName, 0).Result()
	if err != nil {
		return StreamInfo{}, err
	}

	groups := make([]StreamConsumerGroup, 0, len(stream.Groups))
	for _, group := range stream.Groups {
		hydratedGroup, err := hydrateConsumerGroup(ctx, c.rdb, streamName, group)
		if err != nil {
			return StreamInfo{}, err
		}
		groups = append(groups, hydratedGroup)
	}

	return StreamInfo{
		Name:   streamName,
		Groups: groups,
	}, nil
}

func (c *RedisClient) DeleteKey(ctx context.Context, key ...string) error {
	return c.rdb.Del(ctx, key...).Err()
}

func (c *RedisClient) WipeAckMeta(ctx context.Context, streamName string) error {
	ackKey := fmt.Sprintf(ackReportKeyTmpl, streamName)
	return c.DeleteKey(ctx, ackKey)
}

func (c *RedisClient) DeleteKeyWithPattern(ctx context.Context, pattern string) error {
	var cur uint64
	for {
		keys, nextCur, err := c.rdb.Scan(ctx, cur, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.DeleteKey(ctx, keys...); err != nil {
				return err
			}
		}
		cur = nextCur
		if cur == 0 {
			return nil
		}
	}
}

func (c *RedisClient) GetConsumerGroups(ctx context.Context, streamName string) ([]StreamConsumerGroup, error) {
	groups, err := getConsumerGroups(ctx, c.rdb, streamName)
	if err != nil {
		return nil, err
	}

	streamGroups := make([]StreamConsumerGroup, len(groups))
	for _, g := range groups {
		group := StreamConsumerGroup{
			Name:    g.Name,
			Pending: g.Pending,
		}
		streamGroups = append(streamGroups, group)
	}
	return streamGroups, nil
}

func (c *RedisClient) GetAllEvents(ctx context.Context, streamName string) ([]StreamEventInfo, error) {
	events, err := c.rdb.XRevRange(ctx, streamName, "+", "-").Result()
	if err != nil {
		return nil, err
	}

	var eventsInfo []StreamEventInfo
	for _, event := range events {
		ackValue, err := getAckStatus(ctx, c.rdb, streamName, event)
		if err != nil {
			return nil, err
		}

		info := StreamEventInfo{
			ID:        event.ID,
			Values:    event.Values,
			Consumers: ackValue,
		}
		eventsInfo = append(eventsInfo, info)
	}

	return eventsInfo, nil
}

func (c *RedisClient) DeleteAckMeta(ctx context.Context, streamName, id string) error {
	key := fmt.Sprintf(ackReportKeyTmpl, streamName)
	pattern := fmt.Sprintf("%s:*", id)

	var cur uint64
	for {
		fields, nextCur, err := c.rdb.HScanNoValues(ctx, key, cur, pattern, 0).Result()
		if err != nil {
			return err
		}

		if len(fields) > 0 {
			if err := c.rdb.HDel(ctx, key, fields...).Err(); err != nil {
				return err
			}
		}

		cur = nextCur
		if cur == 0 {
			break
		}
	}

	return nil
}

func (c *RedisClient) DeleteConsumer(ctx context.Context, streamName, group, consumer string) error {
	if err := c.rdb.XGroupDelConsumer(ctx, streamName, group, consumer).Err(); err != nil {
		return err
	}

	streamsKey := fmt.Sprintf(streamsKeyTmpl, group, consumer)
	if err := c.rdb.SRem(ctx, streamsKey, streamName).Err(); err != nil {
		return err
	}

	return nil
}
