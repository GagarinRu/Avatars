package queue

const (
	ExchangeDirect = "avatars.direct"
	ExchangeDLX    = "avatars.dlx"

	QueueProcessing = "avatars.processing"
	QueueRetry30s   = "avatars.retry.30s"
	QueueRetry2m    = "avatars.retry.2m"
	QueueDLQ        = "avatars.dlq"

	RoutingUpload = "avatar.upload"
	RoutingRetry  = "retry"
	RoutingDLQ    = "dlq"
)
