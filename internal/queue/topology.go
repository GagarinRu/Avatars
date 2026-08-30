package queue

import amqp "github.com/rabbitmq/amqp091-go"

func Declare(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(ExchangeDLX, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(ExchangeDirect, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(QueueProcessing, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueProcessing, RoutingRetry, ExchangeDLX, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueProcessing, RoutingUpload, ExchangeDirect, false, nil); err != nil {
		return err
	}

	retry30Args := amqp.Table{
		"x-message-ttl":             int32(30_000),
		"x-dead-letter-exchange":    ExchangeDLX,
		"x-dead-letter-routing-key": RoutingRetry,
	}
	if _, err := ch.QueueDeclare(QueueRetry30s, true, false, false, false, retry30Args); err != nil {
		return err
	}

	retry2mArgs := amqp.Table{
		"x-message-ttl":             int32(120_000),
		"x-dead-letter-exchange":    ExchangeDLX,
		"x-dead-letter-routing-key": RoutingRetry,
	}
	if _, err := ch.QueueDeclare(QueueRetry2m, true, false, false, false, retry2mArgs); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(QueueDLQ, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueDLQ, RoutingDLQ, ExchangeDLX, false, nil); err != nil {
		return err
	}

	return nil
}
