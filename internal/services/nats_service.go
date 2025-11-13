package services

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

var nc *nats.Conn

// ConnectNATS Connect initializes the NATS connection
func ConnectNATS(url string) (*nats.Conn, error) {
	var err error
	nc, err = nats.Connect(url,
		nats.MaxReconnects(-1), // retry forever
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to NATS at", url)
	return nc, nil
}

// PublishNATS Publish sends a message to a subject
func PublishNATS(subject string, msg []byte) error {
	if nc == nil || !nc.IsConnected() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, msg)
}

// SubscribeNATS Subscribe listens to a subject with a handler
func SubscribeNATS(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if nc == nil || !nc.IsConnected() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Subscribe(subject, handler)
}
