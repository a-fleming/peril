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
	fmt.Println("Starting Peril server...")
	connectionStr := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionStr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer connection.Close()
	fmt.Println("Successfully established connection to RabbitMQ")

	channel, err := connection.Channel()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// wait for Ctrl-C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	gamelogic.PrintServerHelp()
	replRunning := true
	for replRunning { // start REPL loop
		userInput := gamelogic.GetInput()
		if len(userInput) == 0 {
			continue
		}
		cmd := userInput[0]
		switch cmd {
		case "pause":
			err = publishPauseMessage(channel, true)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			} else {
				fmt.Println("published pause message")
			}
		case "resume":
			err = publishPauseMessage(channel, false)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			} else {
				fmt.Println("published resume message")
			}
		case "quit":
			fmt.Println("exitting REPL loop")
			replRunning = false
			err = sendInterruptSignal()
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
		default:
			fmt.Printf("Unknown command received: '%s'\n", cmd)
		}
	}

	<-signalChan
	fmt.Println("Shutting down Peril server...")
	connection.Close()
	os.Exit(0)
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

func sendInterruptSignal() error {
	pid := os.Getpid()
	return syscall.Kill(pid, syscall.SIGINT)
}
