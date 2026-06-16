package main

import (
	"embed"
	"event-manager/pubsub"
	"event-manager/route"
	"io/fs"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "github.com/supakarn-j/stream-go"
)

//go:embed frontend/dist frontend/dist/assets/*
var staticFiles embed.FS

func main() {
	app := initApp()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	defer rdb.Close()

	producer, err := stream.NewProducer(
		stream.ProducerConfig{MaxLen: stream.DefaultProducerMaxLen},
		stream.WithNewRedisClient(stream.RedisConfig{
			Addr: "localhost:6379",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	r := gin.Default()

	distFS, err := fs.Sub(staticFiles, "frontend/dist")
	if err != nil {
		panic(err)
	}

	listener := pubsub.NewRedisPubSubListener(rdb)

	route.NewRoute(r, listener, rdb, producer, distFS).
		Register()

	listener.Listen()

	app.logger.Infof("[ENV: %s] Starting server on port %s...", app.Vars.ENV, app.Vars.PORT)
	if err := r.Run(":" + app.Vars.PORT); err != nil {
		app.logger.Fatalf("Failed to start server: %v", err)
	}
}
