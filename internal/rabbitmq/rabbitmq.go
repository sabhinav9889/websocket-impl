// interface.go
package rabbitmq

import (
    "log"
    "github.com/streadway/amqp"
)

type RabbitMQInterface interface {
    Publish(queueName string, message string) error
    Consume(queueName string, handler func(string)) error
}


type RabbitMQ struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func NewRabbitMQ(url string) *RabbitMQ {
    conn, err := amqp.Dial(url)
    if err != nil {
        log.Fatalf("Failed to connect to RabbitMQ: %v", err)
    }
    channel, err := conn.Channel()
    if err != nil {
        log.Fatalf("Failed to open a channel: %v", err)
    }
    return &RabbitMQ{
        conn:    conn,
        channel: channel,
    }
}

func (r *RabbitMQ) Publish(queueName string, message string) error {
    _, err := r.channel.QueueDeclare(
        queueName,
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    return r.channel.Publish(
        "",
        queueName,
        false,
        false,
        amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(message),
        },
    )
}

func (r *RabbitMQ) Consume(queueName string, handler func(string)) error {
    msgs, err := r.channel.Consume(
        queueName,
        "",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return err
    }
    go func() {
        for msg := range msgs {
            handler(string(msg.Body))
        }
    }()
    return nil
}
