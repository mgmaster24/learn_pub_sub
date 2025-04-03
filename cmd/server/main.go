package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"

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

	pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("Ctrl+C recieved. Exiting...")
	os.Exit(0)
}
