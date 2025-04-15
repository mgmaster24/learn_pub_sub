package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	fmt.Println("Starting Peril client...")
	connection := "amqp://guest:guest@localhost:5672/"
	c, err := amqp.Dial(connection)
	if err != nil {
		log.Fatalf("Failed to create rabbitmq connection. err: %e", err)
		os.Exit(1)
	}

	defer c.Close()

	publishCh, err := c.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username")
	}

	units := []string{"infantry", "cavalry", "artillery"}
	locations := []string{"americas", "europe", "africa", "asia", "antarctica", "australia"}
	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		c,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, gameState.GetUsername()),
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, "*"),
		int(pubsub.Transient),
		handlerMove(gameState, publishCh),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to army_moves queue")
	}

	err = pubsub.SubscribeJSON(
		c,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix),
		int(pubsub.Durable),
		handlerWar(gameState, publishCh),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to war recognition queue")
	}

	err = pubsub.SubscribeJSON(
		c,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		int(pubsub.Transient),
		handlerPause(gameState),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to pause topic. err: %s", err)
	}
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "spawn":
			if slices.Contains(locations, input[1]) && slices.Contains(units, input[2]) {
				gameState.CommandSpawn(input)
			}
		case "move":
			if slices.Contains(locations, input[1]) {
				if _, err := strconv.Atoi(input[2]); err == nil {
					armyMove, err := gameState.CommandMove(input)
					if err != nil {
						log.Printf("error issuing move command. err: %s", err)
						continue
					}
					err = pubsub.PublishJSON(
						publishCh,
						routing.ExchangePerilTopic,
						fmt.Sprintf("army_moves.%s", username),
						armyMove,
					)
					if err != nil {
						log.Fatalf("Error publishing move, err: %s", err)
						os.Exit(1)
					}
					fmt.Println("move published successfully")
				}
			}
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			return
		case "spam":
			if len(input) < 2 {
				fmt.Println("usage: spam <n>")
				continue
			}

			err = spam(input[1], publishCh, username)
			if err != nil {
				log.Println(err)
				continue
			}
		default:
			fmt.Println("unrecognized command")
		}
	}
}

func spam(num string, publishCh *amqp.Channel, username string) error {
	numTimes, err := strconv.Atoi(num)
	if err != nil {
		fmt.Printf("error: %s is not a valid number\n", num)
		return err
	}

	for range numTimes {
		err = publishGameLog(publishCh, username, gamelogic.GetMaliciousLog())
		if err != nil {
			log.Print("Failed to publish malicious log")
			continue
		}
	}

	return nil
}

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}
