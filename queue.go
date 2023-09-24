package uker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection config struct
type QueueConnData struct {
	Host     string
	User     string
	Password string
}

// Consuming config struct
type QueueConsumeData struct {
	Queue    string
	Exchange string
}

// Queue message struct
type QueueMessage struct {
	Event           string
	Exchange        string
	KeepTrying      bool
	DeliveryMode    uint8
	MessageData     interface{}
	DeadLetterQueue string
}

// Global interface
type Queue interface {
	// SendQueueMessage will send a message via AMQP to the specified exchange
	SendQueueMessage(conn QueueConnData, msg QueueMessage) error
	// ConsumeQueueMessage will consume the messages from specified exchange and queue
	ConsumeQueueMessages(conn QueueConnData, cons QueueConsumeData) (<-chan amqp.Delivery, error)
}

// Local struct to be implmented
type queue_implementation struct{}

// External contructor
func NewQueue() Queue {
	return &queue_implementation{}
}

func (rmq *queue_implementation) ConsumeQueueMessages(conn QueueConnData, cons QueueConsumeData) (<-chan amqp.Delivery, error) {
	var chDelivery <-chan amqp.Delivery
	var err error

	// Set up initial backoff configuration
	backoffCfg := backoff.NewExponentialBackOff()
	backoffCfg.MaxInterval = 5 * time.Minute // Set maximum backoff interval

	// Main loop for connection recovery and backoff
	for {
		// Try reconnecting and setting up consumption with backoff
		chDelivery, err = consumeWithBackoffAndReconnect(conn, cons, backoffCfg)
		if err != nil {
			log.Printf("Error while setting up consumer: %v. Retrying with backoff...", err)
		} else {
			break
		}
	}

	return chDelivery, nil
}

func consumeWithBackoffAndReconnect(conn QueueConnData, cons QueueConsumeData, backoffCfg *backoff.ExponentialBackOff) (<-chan amqp.Delivery, error) {
	var chDelivery <-chan amqp.Delivery
	var rmqConn *amqp.Connection

	for {
		err := backoff.Retry(func() error {
			var err error
			rmqConn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/", conn.User, conn.Password, conn.Host))
			if err != nil {
				return err
			}

			ch, err := rmqConn.Channel()
			if err != nil {
				return err
			}

			queue, err := ch.QueueDeclare(cons.Queue, true, false, false, false, nil)
			if err != nil {
				return err
			}

			err = ch.ExchangeDeclare(cons.Exchange, amqp.ExchangeDirect, true, false, false, false, nil)
			if err != nil {
				return err
			}

			err = ch.QueueBind(cons.Queue, "", cons.Exchange, false, nil)
			if err != nil {
				return err
			}

			chDelivery, err = ch.Consume(queue.Name, "", false, false, false, false, nil)
			if err != nil {
				return err
			}

			// Set up a handler for close notifications
			closeErrChan := make(chan *amqp.Error)
			rmqConn.NotifyClose(closeErrChan)

			// Handle close notifications
			go func() {
				for {
					closeErr := <-closeErrChan
					log.Printf("Connection closed unexpectedly: %v. Reconnecting...", closeErr)

					// Attempt to reconnect
					for {
						log.Println("Attempting to reconnect...")

						// Try to reconnect
						err := rmqReconnect(rmqConn, &chDelivery, conn, cons)
						if err != nil {
							log.Printf("Reconnection failed: %v. Retrying...", err)
							time.Sleep(5 * time.Second)
							continue
						}

						log.Println("Reconnection successful.")
						break
					}
				}
			}()

			return nil
		}, backoffCfg)

		if err != nil {
			log.Printf("Failed to set up consumer with backoff: %v. Retrying...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		return chDelivery, nil
	}
}

func rmqReconnect(rmqConn *amqp.Connection, chDelivery *<-chan amqp.Delivery, conn QueueConnData, cons QueueConsumeData) error {
	// Cerrar la conexión actual
	if err := rmqConn.Close(); err != nil {
		return err
	}

	// Crear una nueva conexión
	newRmqConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/", conn.User, conn.Password, conn.Host))
	if err != nil {
		return err
	}

	// Configurar el canal y la cola nuevamente
	ch, err := newRmqConn.Channel()
	if err != nil {
		return err
	}

	queue, err := ch.QueueDeclare(cons.Queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(cons.Exchange, amqp.ExchangeDirect, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = ch.QueueBind(cons.Queue, "", cons.Exchange, false, nil)
	if err != nil {
		return err
	}

	newChDelivery, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// Update actual connection
	*chDelivery = newChDelivery

	return nil
}

func (rmq *queue_implementation) SendQueueMessage(conn QueueConnData, msg QueueMessage) error {
	if msg.DeliveryMode != amqp.Persistent && msg.DeliveryMode != amqp.Transient {
		panic(fmt.Errorf("invalid amqp message delivery mode %d|%d expected, %d received", amqp.Persistent, amqp.Transient, msg.DeliveryMode))
	}

	// Set a backoff strategy
	boData := backoff.NewExponentialBackOff()
	// Set backoff max interval
	boData.MaxInterval = time.Minute * 5

	// Realiza reintentos con backoff
	err := backoff.Retry(sendQueueMsgOperation(conn, msg), boData)

	if err != nil {
		// If every attempts fail, route message to Dead Letter Queue
		if msg.DeadLetterQueue != "" {
			err = routeMessageToDeadLetter(conn, msg)
			if err != nil {
				return err
			}
		}

		return err
	}

	return nil
}

// Method to send a message via AMQP
func sendQueueMsgOperation(conn QueueConnData, msg QueueMessage) func() error {
	return func() error {
		ch, err := amqpConnectionDial(conn)
		if err != nil {
			return err
		}

		// Declare the exchange
		err = ch.ExchangeDeclare(msg.Exchange, amqp.ExchangeDirect, true, false, false, false, nil)
		if err != nil {
			return err
		}

		// Declare the queue
		queue, err := ch.QueueDeclare(msg.DeadLetterQueue, true, false, false, false, nil)
		if err != nil {
			return err
		}

		// Bind the queue to the exchange
		err = ch.QueueBind(queue.Name, "", msg.Exchange, false, nil)
		if err != nil {
			return err
		}

		// Marshal message content to JSON
		value, err := json.Marshal(msg.MessageData)
		if err != nil {
			return err
		}

		// Try to send the message
		err = ch.PublishWithContext(context.Background(), msg.Exchange, "", false, false, amqp.Publishing{
			Body:         value,
			Headers:      nil,
			ContentType:  "application/json",
			DeliveryMode: msg.DeliveryMode,
		})

		if err != nil {
			return err
		}

		return nil
	}
}

// Método para enrutar un mensaje a la Dead Letter Queue
func routeMessageToDeadLetter(conn QueueConnData, msg QueueMessage) error {
	ch, err := amqpConnectionDial(conn)
	if err != nil {
		return err
	}

	value, err := json.Marshal(msg.MessageData)
	if err != nil {
		return err
	}

	// Try send the message
	err = ch.PublishWithContext(context.Background(), "", msg.DeadLetterQueue, false, false, amqp.Publishing{
		Body:         value,
		Headers:      nil,
		ContentType:  "application/json",
		DeliveryMode: msg.DeliveryMode,
	})

	if err != nil {
		return err
	}

	return nil
}

func amqpConnectionDial(conn QueueConnData) (*amqp.Channel, error) {
	// Try to connect to RabbitMQ
	rmqConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/", conn.User, conn.Password, conn.Host))
	if err != nil {
		return nil, err
	}
	defer rmqConn.Close()

	ch, err := rmqConn.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	return ch, nil
}
