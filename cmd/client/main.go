package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer connection.Close()
	fmt.Println("Successfully established connection to RabbitMQ")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	queueName := routing.PauseKey + "." + username
	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	queueType := pubsub.Transient
	_, _, err = pubsub.DeclareAndBind(connection, exchange, queueName, routingKey, queueType)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// wait for Ctrl-C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	gameState := gamelogic.NewGameState(username)

	replRunning := true
	for replRunning { // start REPL loop
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
				fmt.Printf("%s\n", err)
			} else {
				fmt.Println("move successful")
			}
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			replRunning = false
			err = sendInterruptSignal()
			if err != nil {
				fmt.Printf("%s\n", err)
			}
		default:
			fmt.Printf("Unknown command received: '%s'\n", cmd)
		}
	}

	<-signalChan
	fmt.Println("Shutting down Peril client...")
	connection.Close()
	os.Exit(0)
}

func sendInterruptSignal() error {
	pid := os.Getpid()
	return syscall.Kill(pid, syscall.SIGINT)
}
