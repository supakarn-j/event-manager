package route

import (
	"event-manager/pubsub"

	"github.com/centrifugal/centrifuge"
	"github.com/redis/go-redis/v9"
)

func registerEventListener(listener *pubsub.RedisPubSubListener, rdb *redis.Client, node *centrifuge.Node) {

	handler := newEventHandler(rdb, node)

	listener.RegisterEvent("consumer:status", handler.handleConsumerHealthyEvent)
	listener.RegisterEvent("consumer:ack", handler.handleConsumerAckReport)
	listener.RegisterEvent("__keyevent@0__:xadd", handler.handleNewEventAdded)
	listener.RegisterEvent("__keyevent@0__:hexpired", handler.handleConsumerExpired)
}
