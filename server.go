package main

import (
	"context"
	"embed"
	"encoding/json"
	"event-manager/route"
	"event-manager/utils"
	"fmt"
	"io/fs"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "gitlab.com/hannlync/backend/stream-go.git"
)

//go:embed frontend/dist/*
//go:embed frontend/dist/assets/*
var staticFiles embed.FS

func main() {
	app := initApp()

	r := gin.Default()
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer rdb.Close()

	producer, err := stream.NewProducer(
		stream.ProducerConfig{MaxLen: stream.DefaultProducerMaxLen},
		stream.WithNewRedisClient(stream.RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	distFS, err := fs.Sub(staticFiles, "frontend/dist")
	if err != nil {
		panic(err)
	}

	route.RegisterAPI(r, rdb, producer)
	route.RegisterFrontend(r, rdb, distFS)
	route.RegisterWS(r, rdb, node)

	go func() {
		publishHealth(context.Background(), rdb, node)
	}()

	app.logger.Infof("[ENV: %s] Starting server on port %s...", app.Vars.ENV, app.Vars.PORT)
	if err := r.Run(":" + app.Vars.PORT); err != nil {
		app.logger.Fatalf("Failed to start server: %v", err)
	}
}

func extractConsumer(key string) (string, string, string, bool) {
	parts := strings.SplitN(key, ":", 4)
	if len(parts) != 4 {
		return "", "", "", false
	}

	return fmt.Sprintf("%s:%s", parts[0], parts[1]), parts[2], parts[3], true
}

func publishHealth(ctx context.Context, rdb *redis.Client, node *centrifuge.Node) {
	sub := rdb.PSubscribe(ctx, "__keyevent@0__:hset", "__keyevent@0__:hexpired", "__keyevent@0__:xadd")
	for msg := range sub.Channel() {
		switch msg.Channel {
		case "__keyevent@0__:hset":
			key := msg.Payload
			stream, group, consumer, ok := extractConsumer(key)
			if !ok {
				continue
			}

			res, _ := rdb.HGet(ctx, key, "timestamp").Result()
			var healthy bool
			var lastSeen string
			if res != "" {
				healthy = true
				unix, _ := strconv.ParseInt(res, 10, 64)
				lastSeen = time.Unix(int64(unix), 0).Format("2006-01-02 15:04:05 -07:00 MST")
			}
			payload := map[string]any{
				"action":    "health_check",
				"group":     group,
				"consumer":  consumer,
				"healthy":   healthy,
				"last_seen": lastSeen,
			}
			payloadJson, _ := json.Marshal(payload)
			_, err := node.Publish(stream, payloadJson)
			if err != nil {
				log.Printf("failed to publish health event: %v", err)
				continue
			}
		case "__keyevent@0__:hexpired":
			stream, group, consumer, ok := extractConsumer(msg.Payload)
			if !ok {
				continue
			}

			log.Printf("health check expired for %s:%s:%s", stream, group, consumer)
			payload := map[string]any{
				"action":    "health_check",
				"group":     group,
				"consumer":  consumer,
				"healthy":   false,
				"last_seen": time.Now().Format("2006-01-02 15:04:05 -07:00 MST"),
			}

			payloadJson, _ := json.Marshal(payload)
			_, err := node.Publish(stream, payloadJson)
			if err != nil {
				log.Printf("failed to publish health event: %v", err)
				continue
			}
		case "__keyevent@0__:xadd":
			stream := msg.Payload
			res, err := rdb.XRevRangeN(ctx, stream, "+", "-", 1).Result()
			if err != nil {
				log.Printf("failed to fetch latest message for stream %s: %v", stream, err)
				continue
			}
			if len(res) == 0 {
				log.Printf("xadd event for stream %s had no remaining messages", stream)
				continue
			}

			log.Printf("msg: %+v", res)
			msg := res[0]
			payload := map[string]any{
				"action":    "event_added",
				"timestamp": utils.TimeFromID(msg.ID),
				"id":        msg.ID,
				"values":    msg.Values,
			}
			payloadJson, _ := json.Marshal(payload)
			_, err = node.Publish(stream, payloadJson)
			if err != nil {
				log.Printf("failed to publish new event: %v", err)
				continue
			}
		}
	}
}
