package queue

import amqp "github.com/rabbitmq/amqp091-go"

// Channel is the subset of amqp.Channel used by publisher and consumer.
type Channel interface {
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}
