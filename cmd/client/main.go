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
	fmt.Println("Starting Peril client...")

	connectionStr := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionStr)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer connection.Close()
	fmt.Println("Peril game client connected to RabbitMQ")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}
	gameState := gamelogic.NewGameState(username)

	err = subscribeToPauseQueue(connection, username, gameState)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	for {
		userInput := gamelogic.GetInput()
		if len(userInput) == 0 {
			continue
		}
		cmd := userInput[0]
		switch cmd {
		case "spawn":
			err = gameState.CommandSpawn(userInput)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
		case "move":
			_, err := gameState.CommandMove(userInput)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("move successful")
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unknown command received: '%s'\n", cmd)
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	handler := func(ps routing.PlayingState) {
		defer fmt.Print(">")
		gs.HandlePause(ps)
	}
	return handler
}

func subscribeToPauseQueue(conn *amqp.Connection, username string, gs *gamelogic.GameState) error {
	queueName := routing.PauseKey + "." + username
	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	queueType := pubsub.Transient
	handler := handlerPause(gs)
	return pubsub.SubscribeJSON(conn, exchange, queueName, routingKey, queueType, handler)
}
