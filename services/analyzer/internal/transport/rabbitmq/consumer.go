package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type Consumer struct {
	url           string
	queueName     string
	prefetchCount int
	maxRetries    int
	retryDelay    time.Duration
	logger        *logrus.Logger
	conn          *amqp.Connection
	channel       *amqp.Channel
}

func NewConsumer(url, queueName string, prefetchCount, maxRetries int, retryDelay time.Duration, logger *logrus.Logger) (*Consumer, error) {
	consumer := &Consumer{
		url:           url,
		queueName:     queueName,
		prefetchCount: prefetchCount,
		maxRetries:    maxRetries,
		retryDelay:    retryDelay,
		logger:        logger,
	}

	if err := consumer.connect(); err != nil {
		return nil, err
	}

	return consumer, nil
}

func (c *Consumer) connect() error {
	var err error

	for i := 0; i < c.maxRetries; i++ {
		c.conn, err = amqp.Dial(c.url)
		if err == nil {
			break
		}

		c.logger.WithError(err).Errorf("Failed to connect to RabbitMQ (attempt %d/%d)", i+1, c.maxRetries)
		time.Sleep(c.retryDelay)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", c.maxRetries, err)
	}

	c.channel, err = c.conn.Channel()
	if err != nil {
		if closeErr := c.conn.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Error("Failed to close connection during cleanup")
		}
		return fmt.Errorf("failed to open channel: %w", err)
	}

	_, err = c.channel.QueueDeclare(
		c.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		if closeErr := c.conn.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Error("Failed to close connection during cleanup")
		}
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = c.channel.Qos(
		c.prefetchCount, // prefetch count
		0,               // prefetch size
		false,           // global
	)
	if err != nil {
		if closeErr := c.conn.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Error("Failed to close connection during cleanup")
		}
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	go func() {
		<-c.conn.NotifyClose(make(chan *amqp.Error))
		c.logger.Warn("RabbitMQ connection closed, attempting to reconnect...")
		for {
			if err := c.connect(); err != nil {
				c.logger.WithError(err).Error("Failed to reconnect to RabbitMQ")
				time.Sleep(c.retryDelay)
				continue
			}
			c.logger.Info("Successfully reconnected to RabbitMQ")
			break
		}
	}()

	return nil
}

func (c *Consumer) Consume(ctx context.Context, handler func(message []byte) error) error {
	if c.conn == nil || c.conn.IsClosed() {
		return errors.New("connection is not open")
	}

	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("Consumer channel closed")
				return nil
			}

			c.logger.WithFields(logrus.Fields{
				"delivery_tag": msg.DeliveryTag,
				"content_type": msg.ContentType,
			}).Debug("Received message")

			err := handler(msg.Body)
			if err != nil {
				c.logger.WithError(err).Error("Failed to process message")
				if err := msg.Nack(false, true); err != nil {
					c.logger.WithError(err).Error("Failed to nack message")
				}
			} else {
				if err := msg.Ack(false); err != nil {
					c.logger.WithError(err).Error("Failed to ack message")
				}
			}
		}
	}
}

func (c *Consumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}
