package route

import (
	"log"

	"github.com/centrifugal/centrifuge"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterWS(r *gin.Engine, rdb *redis.Client, node *centrifuge.Node) {
	node.OnConnect(func(client *centrifuge.Client) {
		transportName := client.Transport().Name()
		transportProto := client.Transport().Protocol()

		log.Printf("client connected via %s (%s)", transportName, transportProto)
		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			cb(centrifuge.SubscribeReply{}, nil)
		})

		client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			log.Printf("client disconnected: %s", client.ID())
		})
	})

	if err := node.Run(); err != nil {
		log.Fatal(err)
	}

	wsHandler := centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{})

	r.GET("/ws", func(c *gin.Context) {
		ctx := centrifuge.SetCredentials(c.Request.Context(), &centrifuge.Credentials{
			UserID: "", // empty = anonymous
		})

		req := c.Request.WithContext(ctx)
		wsHandler.ServeHTTP(c.Writer, req)
	})

}
