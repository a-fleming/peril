package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Transient SimpleQueueType = iota
	Durable
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
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
	args := amqp.Table{
		"x-dead-letter-exchange": routing.ExchangePerilDeadLetter,
	}
	if queueType == Durable {
		durable = true
		autoDelete = false
		exclusive = false
	}
	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, args)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	err = channel.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return channel, queue, nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	})
}

func PublishJson[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonBytes,
	})
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	err = channel.Qos(10, 0, false)
	if err != nil {
		return err
	}
	deliveryChannel, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for delivery := range deliveryChannel {
			msgBytes, err := unmarshaller(delivery.Body)
			if err != nil {
				log.Println(err)
				continue
			}
			ackType := handler(msgBytes)
			switch ackType {
			case Ack:
				err = delivery.Ack(false)
				log.Println("Ack sent")
			case NackRequeue:
				err = delivery.Nack(false, true)
				log.Println("NackRequeue sent")
			case NackDiscard:
				err = delivery.Nack(false, false)
				log.Println("NackDiscard sent")
			default:
				log.Printf("Unknown ackType received: %v\n", ackType)
				continue
			}
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}()
	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	deliveryChannel, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for delivery := range deliveryChannel {
			var msgBytes T
			err = json.Unmarshal(delivery.Body, &msgBytes)
			if err != nil {
				log.Println(err)
				continue
			}
			ackType := handler(msgBytes)
			switch ackType {
			case Ack:
				err = delivery.Ack(false)
				log.Println("Ack sent")
			case NackRequeue:
				err = delivery.Nack(false, true)
				log.Println("NackRequeue sent")
			case NackDiscard:
				err = delivery.Nack(false, false)
				log.Println("NackDiscard sent")
			default:
				log.Printf("Unknown ackType received: %v\n", ackType)
				continue
			}
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}()
	return nil
}
