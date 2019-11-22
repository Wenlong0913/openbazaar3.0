package core

import (
	"context"
	"github.com/cpacia/openbazaar3.0/events"
	"testing"
	"time"
)

func TestOpenBazaarNode_PingNode(t *testing.T) {
	network, err := NewMocknet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer network.TearDown()

	pingSub, err := network.Nodes()[1].eventBus.Subscribe(&events.PingReceived{})
	if err != nil {
		t.Fatal(err)
	}
	pongSub, err := network.Nodes()[0].eventBus.Subscribe(&events.PongReceived{})
	if err != nil {
		t.Fatal(err)
	}
	if err := network.Nodes()[0].PingNode(context.Background(), network.Nodes()[1].Identity()); err != nil {
		t.Fatal(err)
	}
	select {
	case i := <-pingSub.Out():
		event := i.(*events.PingReceived)
		if event.Peer.Pretty() != network.Nodes()[0].Identity().Pretty() {
			t.Error("Event contained incorrect peerID")
		}
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
	select {
	case i := <-pongSub.Out():
		event := i.(*events.PongReceived)
		if event.Peer.Pretty() != network.Nodes()[1].Identity().Pretty() {
			t.Error("Event contained incorrect peerID")
		}
	case <-time.After(time.Second * 10):
		t.Fatal("Timeout waiting on channel")
	}
}
