package messaging

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

// RabbitMQInterface defines the interface for RabbitMQ operations
type RabbitMQInterface interface {
	DeclareQueue(name string) (interface{}, error)
	DeclareQueueWithDLQ(name string, dlqName string) (interface{}, error)
	PublishMessage(queueName string, message interface{}) error
	PublishMessageWithHeaders(queueName string, message interface{}, headers map[string]interface{}) error
	ConsumeMessages(queueName string, handler func([]byte) error) error
	GetMessages(queueName string, maxMessages int) ([]amqp.Delivery, error)
	RequeueMessage(queueName string, message []byte) error
	Close()
}

// RabbitMQ queues
const (
	QueueImageUpload       = "image_upload"
	QueueMetadataGenerated = "metadata_generated"
	QueueImageProcess      = "image_process"
	QueueImageProcessed    = "image_processed"
	QueueDeadLetter        = "dead_letter_queue"
)

// RabbitMQClient handles messaging with RabbitMQ
type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQClient creates a new RabbitMQ client
func NewRabbitMQClient(url string) (*RabbitMQClient, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing connection: %v", closeErr)
		}
		return nil, err
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: channel,
	}, nil
}

// Close closes the connection and channel
func (c *RabbitMQClient) Close() {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
}

// DeclareQueue declares a queue with the given name
func (c *RabbitMQClient) DeclareQueue(name string) (interface{}, error) {
	return c.channel.QueueDeclare(
		name,  // queue name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

// DeclareQueueWithDLQ declares a queue with dead letter queue support
func (c *RabbitMQClient) DeclareQueueWithDLQ(name string, dlqName string) (interface{}, error) {
	// First, declare the dead letter queue
	_, err := c.channel.QueueDeclare(
		dlqName, // queue name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}

	// Then declare the main queue with DLQ configuration
	return c.channel.QueueDeclare(
		name,  // queue name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		},
	)
}

// PublishMessage publishes a message to the given queue
func (c *RabbitMQClient) PublishMessage(queueName string, message interface{}) error {
	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish message
	return c.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// PublishMessageWithHeaders publishes a message with custom headers to the given queue
func (c *RabbitMQClient) PublishMessageWithHeaders(queueName string, message interface{}, headers map[string]interface{}) error {
	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Convert headers to amqp.Table
	amqpHeaders := make(amqp.Table)
	for k, v := range headers {
		amqpHeaders[k] = v
	}

	// Publish message
	return c.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     amqpHeaders,
		},
	)
}

// ConsumeMessages consumes messages from the given queue
func (c *RabbitMQClient) ConsumeMessages(queueName string, handler func([]byte) error) error {
	// Get messages from queue
	msgs, err := c.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	// Process messages
	go func() {
		for msg := range msgs {
			log.Printf("Received message from queue: %s", queueName)

			// Process message
			err := handler(msg.Body)
			if err != nil {
				log.Printf("Error processing message: %v", err)
				if nackErr := msg.Nack(false, true); nackErr != nil {
					log.Printf("Error sending NACK: %v", nackErr)
				}
			} else {
				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("Error sending ACK: %v", ackErr)
				}
			}
		}
	}()

	return nil
}

// GetMessages retrieves messages from a queue without consuming them permanently
func (c *RabbitMQClient) GetMessages(queueName string, maxMessages int) ([]amqp.Delivery, error) {
	messages := make([]amqp.Delivery, 0, maxMessages)

	for i := 0; i < maxMessages; i++ {
		msg, ok, err := c.channel.Get(queueName, false)
		if err != nil {
			return nil, err
		}
		if !ok {
			// No more messages
			break
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// RequeueMessage publishes a message back to the original queue
func (c *RabbitMQClient) RequeueMessage(queueName string, message []byte) error {
	return c.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
}
