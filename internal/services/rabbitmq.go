package services

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/streadway/amqp"
)

type RabbitMQService struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

var rabbitmqInstance *RabbitMQService

type FileProcessingMessage struct {
    FileID      string `json:"file_id"`
    FilePath    string `json:"file_path"`
    FileType    string `json:"file_type"`
    Operation   string `json:"operation"` // "preview", "delete", "process"
}

func InitializeRabbitMQ(connectionString string) error {
    conn, err := amqp.Dial(connectionString)
    if err != nil {
        return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
    }

    channel, err := conn.Channel()
    if err != nil {
        return fmt.Errorf("failed to open channel: %v", err)
    }

    // Declare the queue
    _, err = channel.QueueDeclare(
        "file_processing", // queue name
        true,              // durable
        false,             // delete when unused
        false,             // exclusive
        false,             // no-wait
        nil,               // arguments
    )
    if err != nil {
        return fmt.Errorf("failed to declare queue: %v", err)
    }

    rabbitmqInstance = &RabbitMQService{
        conn:    conn,
        channel: channel,
    }

    log.Println("Connected to RabbitMQ successfully")
    return nil
}

func GetRabbitMQService() *RabbitMQService {
    return rabbitmqInstance
}

func (r *RabbitMQService) PublishFileProcessingMessage(message FileProcessingMessage) error {
    body, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %v", err)
    }

    err = r.channel.Publish(
        "",                 // exchange
        "file_processing", // routing key
        false,              // mandatory
        false,              // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
            DeliveryMode: amqp.Persistent, // Make message persistent
        },
    )

    if err != nil {
        return fmt.Errorf("failed to publish message: %v", err)
    }

    log.Printf("Published message for file: %s, operation: %s", message.FileID, message.Operation)
    return nil
}

func (r *RabbitMQService) Close() {
    if r.channel != nil {
        r.channel.Close()
    }
    if r.conn != nil {
        r.conn.Close()
    }
}