package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func hydrateConsumerGroup(ctx context.Context, rdb *redis.Client, streamName string, group redis.XInfoStreamGroup) (StreamConsumerGroup, error) {
	pendingCount, err := getGroupPendingCount(ctx, rdb, streamName, group.Name)
	if err != nil {
		return StreamConsumerGroup{}, err
	}

	consumers := make([]StreamConsumer, 0, len(group.Consumers))
	for _, consumer := range group.Consumers {
		c := getConsumerHealthStatus(ctx, rdb, streamName, group.Name, consumer)
		consumers = append(consumers, c)
	}

	hydratedGroup := StreamConsumerGroup{
		Name:      group.Name,
		Pending:   pendingCount,
		Consumers: consumers,
	}
	return hydratedGroup, nil
}

func getGroupPendingList(ctx context.Context, rdb *redis.Client, streamName, group string) (*redis.XPending, error) {
	pending, err := rdb.XPending(ctx, streamName, group).Result()
	if err != nil {
		return nil, err
	}

	return pending, nil
}

func getGroupPendingCount(ctx context.Context, rdb *redis.Client, streamName, group string) (int64, error) {
	var count int64
	pending, err := getGroupPendingList(ctx, rdb, streamName, group)
	if err != nil {
		return 0, err
	}

	if pending != nil {
		count = pending.Count
	}

	return count, nil
}

func getConsumerHealthStatus(ctx context.Context, rdb *redis.Client, streamName, group string, consumer redis.XInfoStreamConsumer) StreamConsumer {
	var healthy bool
	var consumerPending int64
	// if pending != nil {
	// 	consumerPending = pending.Consumers[consumer.Name]
	// }

	lastSeen := consumer.SeenTime.Format("2006-01-02 15:04:05 -07:00 MST")
	key := fmt.Sprintf(consumerKeyTmpl, group, consumer.Name)
	seen, _ := rdb.HGet(ctx, key, "lastSeen").Result()
	if seen != "" {
		healthy = true
		lastSeen = seen
	}

	ip, _ := rdb.HGet(ctx, key, "ip").Result()

	res := StreamConsumer{
		Name:     consumer.Name,
		IP:       ip,
		LastSeen: lastSeen,
		Healthy:  healthy,
		Pending:  consumerPending,
	}

	return res
}

func getAckStatus(ctx context.Context, rdb *redis.Client, streamName string, event redis.XMessage) ([]ConsumerAckReport, error) {
	var cur uint64
	var reports []ConsumerAckReport

	key := fmt.Sprintf(ackReportKeyTmpl, streamName)
	pattern := fmt.Sprintf("%s:*", event.ID)

	for {
		values, nextCur, err := rdb.HScan(ctx, key, cur, pattern, 0).Result()
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(values); i += 2 {
			raw := values[i+1]
			var report ConsumerAckReport
			if err := json.Unmarshal([]byte(raw), &report); err != nil {
				continue
			}

			reports = append(reports, report)
		}

		cur = nextCur
		if cur == 0 {
			break
		}
	}

	return reports, nil
}

func getConsumerGroups(ctx context.Context, rdb *redis.Client, stream string) ([]redis.XInfoGroup, error) {
	groups, err := rdb.XInfoGroups(ctx, stream).Result()
	if err != nil {
		return nil, err
	}

	return groups, nil
}
