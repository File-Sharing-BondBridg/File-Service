package services

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

var (
	nc  *nats.Conn
	js  nats.JetStreamContext
	njm nats.JetStreamContext
)

// ConnectNATS connects to NATS and initializes JetStream and streams.
// It returns the underlying Conn and JetStreamContext for advanced usage.
func ConnectNATS(url string) (*nats.Conn, nats.JetStreamContext, error) {
	if nc != nil && nc.IsConnected() {
		return nc, js, nil
	}

	opts := []nats.Option{
		nats.Name("file-service"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("[NATS] disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[NATS] reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("[NATS] connection closed")
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, err
	}
	nc = conn

	// Create JetStream context
	jsCtx, err := nc.JetStream()
	if err != nil {
		// In case JetStream not enabled on server, fail explicitly
		nc.Close()
		nc = nil
		return nil, nil, err
	}
	js = jsCtx
	njm = jsCtx

	// Ensure the stream(s) exist (idempotent)
	if err := ensureStreams(); err != nil {
		log.Printf("[NATS] warning: failed to ensure streams: %v", err)
		// Not fatal here — you may still want to continue without JetStream
	}

	log.Println("[NATS] connected and JetStream initialized")
	return nc, js, nil
}

// ensureStreams creates streams used by the app if they don't exist
func ensureStreams() error {
	_, err := js.StreamInfo("file-events")
	if err == nil {
		log.Printf("[NATS] stream %s already exists", "file-events")
		return nil
	}

	streamCfg := &nats.StreamConfig{
		Name:     "file-events",
		Subjects: []string{"files.*", "users.*"},
		Storage:  nats.FileStorage,
		MaxAge:   30 * 24 * time.Hour,
	}

	_, err = js.AddStream(streamCfg)
	return err
}

// PublishEvent publishes an event via JetStream (durable, stored).
// subject e.g. "files.uploaded"
func PublishEvent(subject string, payload interface{}) error {
	if js == nil {
		return errors.New("jetstream not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use a message ID for idempotency
	msgID := uuid.New().String()
	_, err = js.Publish(subject, data, nats.MsgId(msgID))
	if err != nil {
		log.Printf("[NATS] publish failed subject=%s err=%v", subject, err)
		return err
	}
	return nil
}

// SubscribeEvent creates a durable, ack-based consumer.
// handler receives the nats.Msg and is responsible to Ack() when done.
// durableName should be unique per consumer service (e.g., "file-service-preview-consumer")
func SubscribeEvent(subject, durableName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if js == nil {
		return nil, errors.New("jetstream not initialized")
	}
	// Create a durable, manual-ack subscription (pull or push? use push for simplicity)
	sub, err := js.Subscribe(subject, func(msg *nats.Msg) {
		// wrap handler to provide safe ack / nack handling
		handler(msg)
		// Note: handler should Ack() on success; otherwise implement timeout/nack logic
	}, nats.Durable(durableName), nats.ManualAck())
	if err != nil {
		return nil, err
	}
	log.Printf("[NATS] subscribed (jetstream) subject=%s durable=%s", subject, durableName)
	return sub, nil
}

// PublishPlain publishes without JetStream (fire-and-forget).
// Keep this for compatibility, but prefer PublishEvent for critical events.
func PublishPlain(subject string, payload []byte) error {
	if nc == nil || !nc.IsConnected() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, payload)
}
