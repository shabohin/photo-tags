module github.com/shabohin/photo-tags/services/processor

go 1.24.0

replace github.com/shabohin/photo-tags/pkg => ../../pkg

require github.com/shabohin/photo-tags/pkg v0.0.0

require (
	github.com/minio/minio-go/v7 v7.0.87
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.10.0
)
