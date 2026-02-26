package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connectionStr := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionStr)
	if err != nil {
		log.Fatalf("could not conenct to RabbitMQ: %v", err)
	}
	defer connection.Close()
	fmt.Println("Peril game server connected to RabbitMQ")

	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	queueName := "game_logs"
	exchange := routing.ExchangePerilTopic
	routingKey := routing.GameLogSlug + ".*"
	queueType := pubsub.Durable
	_, queue, err := pubsub.DeclareAndBind(connection, exchange, queueName, routingKey, queueType)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	gamelogic.PrintServerHelp()
	for {
		userInput := gamelogic.GetInput()
		if len(userInput) == 0 {
			continue
		}
		cmd := userInput[0]
		switch cmd {
		case "pause":
			fmt.Println("publishing pause game state")
			err = publishPauseMessage(channel, true)
			if err != nil {
				log.Printf("could not publish pause game state: %v", err)
			}
		case "resume":
			fmt.Println("publishing resume game state")
			err = publishPauseMessage(channel, false)
			if err != nil {
				log.Printf("could not publish resume game state: %v", err)
			}
		case "quit":
			fmt.Println("exitting REPL loop")
			return
		default:
			fmt.Printf("Unknown command received: '%s'\n", cmd)
		}
	}
}

func publishPauseMessage(ch *amqp.Channel, isPaused bool) error {

	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	val := routing.PlayingState{
		IsPaused: isPaused,
	}
	err := pubsub.PublishJson(ch, exchange, routingKey, val)
	if err != nil {
		return err
	}
	return nil
}
