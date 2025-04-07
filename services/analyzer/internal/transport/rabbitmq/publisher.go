package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type Publisher struct {
	url        string
	queueName  string
	maxRetries int
	retryDelay time.Duration
	logger     *logrus.Logger
	conn       *amqp.Connection
	channel    *amqp.Channel
}

func NewPublisher(url, queueName string, maxRetries int, retryDelay time.Duration, logger *logrus.Logger) (*Publisher, error) {
	publisher := &Publisher{
		url:        url,
		queueName:  queueName,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		logger:     logger,
	}

	if err := publisher.connect(); err != nil {
		return nil, err
	}

	return publisher, nil
}

func (p *Publisher) connect() error {
	var err error

	for i := 0; i < p.maxRetries; i++ {
		p.conn, err = amqp.Dial(p.url)
		if err == nil {
			break
		}

		p.logger.WithError(err).Errorf("Failed to connect to RabbitMQ (attempt %d/%d)", i+1, p.maxRetries)
		time.Sleep(p.retryDelay)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", p.maxRetries, err)
	}

	p.channel, err = p.conn.Channel()
	if err != nil {
		p.conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	_, err = p.channel.QueueDeclare(
		p.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		p.conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	go func() {
		<-p.conn.NotifyClose(make(chan *amqp.Error))
		p.logger.Warn("RabbitMQ connection closed, attempting to reconnect...")
		for {
			if err := p.connect(); err != nil {
				p.logger.WithError(err).Error("Failed to reconnect to RabbitMQ")
				time.Sleep(p.retryDelay)
				continue
			}
			p.logger.Info("Successfully reconnected to RabbitMQ")
			break
		}
	}()

	return nil
}

func (p *Publisher) Publish(ctx context.Context, message []byte) error {
	if p.conn == nil || p.conn.IsClosed() {
		if err := p.connect(); err != nil {
			return fmt.Errorf("failed to reconnect before publishing: %w", err)
		}
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := p.channel.PublishWithContext(
		publishCtx,
		"",          // exchange
		p.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)

	if err != nil {
		p.logger.WithError(err).Error("Failed to publish message")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.WithField("queue", p.queueName).Debug("Message published successfully")
	return nil
}

func (p *Publisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			return err
		}
	}

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}
