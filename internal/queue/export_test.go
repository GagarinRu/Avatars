package queue

// ResolveRetryTargetForTest exposes retry routing for unit tests.
func ResolveRetryTargetForTest(retryCount int) (queueName, exchange, routingKey string) {
	return resolveRetryTarget(retryCount)
}
