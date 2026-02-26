package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Transient SimpleQueueType = iota
	Durable
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	durable := false
	autoDelete := true
	exclusive := true
	noWait := false
	if queueType == Durable {
		durable = true
		autoDelete = false
		exclusive = false
	}
	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	err = channel.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return channel, queue, nil
}

func PublishJson[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	ctx := context.Background()
	ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonBytes,
	})
	return nil
}
