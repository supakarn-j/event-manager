package route

import (
	"reflect"
	"testing"

	"github.com/centrifugal/centrifuge"
)

func TestNodeSetupConfiguresProvidedNode(t *testing.T) {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		t.Fatal(err)
	}

	nodeSetup(node)

	if !hasConnectHandler(node) {
		t.Fatal("expected nodeSetup to register connect handler on provided node")
	}
}

func hasConnectHandler(node *centrifuge.Node) bool {
	nodeValue := reflect.ValueOf(node).Elem()
	clientEvents := nodeValue.FieldByName("clientEvents")
	connectHandler := clientEvents.Elem().FieldByName("connectHandler")
	return !connectHandler.IsNil()
}
