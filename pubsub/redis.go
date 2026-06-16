package pubsub

import (
	"context"
	"event-manager/utils"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisPubSubListener struct {
	rdb     *redis.Client
	handler map[string]HandlerFunc
}

type HandlerFunc func(string)

func NewRedisPubSubListener(rdb *redis.Client) *RedisPubSubListener {
	return &RedisPubSubListener{
		rdb:     rdb,
		handler: make(map[string]HandlerFunc),
	}
}

func (l *RedisPubSubListener) RegisterEvent(ch string, handler HandlerFunc) {
	l.handler[ch] = handler
}

func (l *RedisPubSubListener) Listen() {
	go func() {
		l.listenEvents(context.Background())
	}()
}

func (l *RedisPubSubListener) listenEvents(ctx context.Context) {
	listeningCh := utils.ExtractMapKeys(l.handler)
	log.Printf("listening to events: %s", listeningCh)
	sub := l.rdb.PSubscribe(ctx, listeningCh...)
	ch := sub.Channel()

	for msg := range ch {
		if _, ok := l.handler[msg.Channel]; !ok {
			log.Printf("No handler for event %s, skipping", msg.Channel)
			continue
		}

		l.handler[msg.Channel](msg.Payload)
	}
}
