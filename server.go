package main

import (
	"context"
	"embed"
	"event-manager/route"
	"io/fs"
	"log"

	"github.com/centrifugal/centrifuge"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "github.com/supakarn-j/stream-go"
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
		listenEvents(context.Background(), rdb, node)
	}()

	app.logger.Infof("[ENV: %s] Starting server on port %s...", app.Vars.ENV, app.Vars.PORT)
	if err := r.Run(":" + app.Vars.PORT); err != nil {
		app.logger.Fatalf("Failed to start server: %v", err)
	}
}
