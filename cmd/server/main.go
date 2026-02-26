package main

import (
	"fmt"
	"os"
	"os/signal"

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
	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	val := routing.PlayingState{
		IsPaused: true,
	}
	err = pubsub.PublishJson(channel, exchange, routingKey, val)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("published message")

	// wait for Ctrl-C
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Shutting down Peril server...")
	connection.Close()
	os.Exit(0)
}
