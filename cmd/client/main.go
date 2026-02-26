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
	channel, err := connection.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}
	gameState := gamelogic.NewGameState(username)

	err = subscribeToPauseQueue(connection, gameState)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	err = subscribeToMovesQueue(connection, gameState, channel)
	if err != nil {
		log.Fatalf("could not subscribe to moves: %v", err)
	}
	err = subscribeToWarQueue(connection, gameState)
	if err != nil {
		log.Fatalf("could not subscribe to war: %v", err)
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
			armyMove, err := gameState.CommandMove(userInput)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = publishMove(channel, armyMove)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("move published successfully")
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

func handlerMoves(gs *gamelogic.GameState, ch *amqp.Channel) func(move gamelogic.ArmyMove) pubsub.AckType {
	handler := func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := publishWar(ch, gs, move)
			if err != nil {
				log.Println(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
	return handler
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	handler := func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
	return handler
}

func handlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	handler := func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			log.Printf("Unknown war outcome: %v\n", outcome)
			return pubsub.NackDiscard
		}
	}
	return handler
}

func subscribeToPauseQueue(conn *amqp.Connection, gs *gamelogic.GameState) error {
	queueName := routing.PauseKey + "." + gs.GetUsername()
	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	queueType := pubsub.Transient
	handler := handlerPause(gs)
	return pubsub.SubscribeJSON(conn, exchange, queueName, routingKey, queueType, handler)
}

func subscribeToMovesQueue(conn *amqp.Connection, gs *gamelogic.GameState, ch *amqp.Channel) error {
	queueName := routing.ArmyMovesPrefix + "." + gs.GetUsername()
	exchange := routing.ExchangePerilTopic
	routingKey := routing.ArmyMovesPrefix + ".*"
	queueType := pubsub.Transient
	handler := handlerMoves(gs, ch)
	return pubsub.SubscribeJSON(conn, exchange, queueName, routingKey, queueType, handler)
}

func subscribeToWarQueue(conn *amqp.Connection, gs *gamelogic.GameState) error {
	queueName := routing.WarRecognitionsPrefix
	exchange := routing.ExchangePerilTopic
	routingKey := routing.WarRecognitionsPrefix + ".*"
	queueType := pubsub.Durable
	handler := handlerWar(gs)
	return pubsub.SubscribeJSON(conn, exchange, queueName, routingKey, queueType, handler)
}

func publishMove(ch *amqp.Channel, move gamelogic.ArmyMove) error {
	exchange := routing.ExchangePerilTopic
	routingKey := routing.ArmyMovesPrefix + "." + move.Player.Username
	err := pubsub.PublishJson(ch, exchange, routingKey, move)
	if err != nil {
		return err
	}
	return nil
}

func publishWar(ch *amqp.Channel, gs *gamelogic.GameState, move gamelogic.ArmyMove) error {
	exchange := routing.ExchangePerilTopic
	routingKey := routing.WarRecognitionsPrefix + "." + gs.GetUsername()
	recognitionOfWar := gamelogic.RecognitionOfWar{
		Attacker: move.Player,
		Defender: gs.GetPlayerSnap(),
	}
	err := pubsub.PublishJson(ch, exchange, routingKey, recognitionOfWar)
	if err != nil {
		return err
	}
	return nil
}
