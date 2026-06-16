package route

import (
	"log"

	"github.com/centrifugal/centrifuge"
	"github.com/gin-gonic/gin"
)

func registerWS(r *gin.Engine, node *centrifuge.Node) {
	nodeSetup(node)
	if err := node.Run(); err != nil {
		log.Fatal(err)
		return
	}

	r.GET("/ws", wsHandler(node))
}

func wsHandler(node *centrifuge.Node) func(*gin.Context) {
	handler := centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{})

	return func(c *gin.Context) {
		ctx := centrifuge.SetCredentials(c.Request.Context(), &centrifuge.Credentials{
			UserID: "", // empty = anonymous
		})

		req := c.Request.WithContext(ctx)
		handler.ServeHTTP(c.Writer, req)

	}
}

func nodeSetup(node *centrifuge.Node) {
	node.OnConnect(func(client *centrifuge.Client) {
		transportName := client.Transport().Name()
		transportProto := client.Transport().Protocol()

		log.Printf("client connected via %s (%s)", transportName, transportProto)
		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			log.Printf("client %s subscribes to channel %s", client.ID(), e.Channel)
			cb(centrifuge.SubscribeReply{}, nil)
		})

		client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			log.Printf("client disconnected: %s", client.ID())
		})
	})
}
