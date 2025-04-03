package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	connection := "amqp://guest:guest@localhost:5672/"
	c, err := amqp091.Dial(connection)
	if err != nil {
		log.Fatalf("Failed to create rabbitmq connection. err: %e", err)
		os.Exit(1)
	}

	defer c.Close()

	fmt.Println("Connected to Rabbitmq...")
	usename, err := gamelogic.ClientWelcome()
}:wq
