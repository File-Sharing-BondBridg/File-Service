package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

type Client struct {
	Conn *nats.Conn
}

// SubscribeAll loads all routes once during startup
func (c *Client) SubscribeAll(routes map[string]nats.MsgHandler) error {
	for subject, handler := range routes {
		_, err := c.Conn.Subscribe(subject, handler)
		if err != nil {
			return err
		}
		log.Printf("[NATS] Subscribed to: %s", subject)
	}
	return nil
}
