package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type (
	SimpleQueueType int
	AckType         string
)

const (
	Durable   SimpleQueueType = 0
	Transient SimpleQueueType = 1
)

const (
	Ack         AckType = "Ack"
	NackRequeue         = "NackRequeue"
	NackDiscard         = "NackDiscard"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        valBytes,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        b.Bytes(),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	dur := simpleQueueType == int(Durable)
	table := make(amqp.Table)
	table["x-dead-letter-exchange"] = "peril_dlx"
	queue, err := ch.QueueDeclare(queueName, dur, !dur, !dur, false, table)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshallJSON)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshallGob)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	c, q, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	err = c.Qos(10, 0, false)
	if err != nil {
		return err
	}

	d, err := c.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func(delivery <-chan amqp.Delivery) {
		for msg := range delivery {
			obj, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("Error unmarshaling the message. err:%s\n", err)
				continue
			}
			switch handler(obj) {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}
		}
	}(d)

	return nil
}

func unmarshallJSON[T any](body []byte) (T, error) {
	var obj T
	err := json.Unmarshal(body, &obj)
	return obj, err
}

func unmarshallGob[T any](body []byte) (T, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(body))
	var obj T
	err := dec.Decode(&obj)
	return obj, err
}
