package route

import (
	"event-manager/pubsub"
	"io/fs"
	"log"

	"github.com/centrifugal/centrifuge"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	stream "github.com/supakarn-j/stream-go"
)

type Route struct {
	router   *gin.Engine
	listener *pubsub.RedisPubSubListener
	rdb      *redis.Client
	node     *centrifuge.Node
	producer *stream.Producer
	distFS   fs.FS
}

func NewRoute(r *gin.Engine, listener *pubsub.RedisPubSubListener, rdb *redis.Client, producer *stream.Producer, distFS fs.FS) *Route {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		log.Fatal(err)
	}

	return &Route{
		router:   r,
		listener: listener,
		rdb:      rdb,
		node:     node,
		producer: producer,
		distFS:   distFS,
	}
}

func (r *Route) Register() {
	registerApiRoute(r)
	registerFrontend(r.router, r.rdb, r.distFS)
	registerWS(r.router, r.node)
	registerEventListener(r.listener, r.rdb, r.node)
}
