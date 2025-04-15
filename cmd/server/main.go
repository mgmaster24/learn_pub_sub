package main

import (
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	connection := "amqp://guest:guest@localhost:5672/"
	c, err := amqp.Dial(connection)
	if err != nil {
		log.Fatalf("Failed to create rabbitmq connection. err: %e", err)
		os.Exit(1)
	}

	defer c.Close()

	fmt.Println("Connected to Rabbitmq...")

	ch, err := c.Channel()
	if err != nil {
		return
	}

	err = pubsub.SubscribeGob(
		c,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		int(pubsub.Durable),
		handlerLogs(),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to logs. err: %s", err)
	}

	gamelogic.PrintServerHelp()
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		if input[0] == "pause" {
			err = publishPauseMsg(ch, true)
			if err != nil {
				log.Fatalf("could not pause the game. err: %s", err)
			}
		} else if input[0] == "resume" {
			err = publishPauseMsg(ch, false)
			if err != nil {
				log.Fatalf("could not resume the game. err: %s", err)
			}

		} else if input[0] == "quit" {
			fmt.Println("Quitting the game...")
			return
		} else {
			fmt.Println("Unrecognized Command")
		}
	}
}

func publishPauseMsg(channel *amqp.Channel, pause bool) error {
	return pubsub.PublishJSON(
		channel,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: pause,
		},
	)
}
