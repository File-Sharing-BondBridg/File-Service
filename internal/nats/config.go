package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

type Client struct {
	Conn *nats.Conn
}

func NewClient(url string) (*Client, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	return &Client{
		Conn: conn,
	}, nil
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
