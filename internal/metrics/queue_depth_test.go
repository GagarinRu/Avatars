package metrics

import "testing"

func TestRabbitManagementBaseURL(t *testing.T) {
	t.Parallel()
	got, err := rabbitManagementBaseURL(RabbitMQManagement{
		Host: "avatars.rabbitmq",
		Port: "15672",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://avatars.rabbitmq:15672"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRabbitMQManagementFromURL(t *testing.T) {
	t.Parallel()
	got, err := RabbitMQManagementFromURL(
		"amqp://guest:guest@avatars.rabbitmq:5672/",
		"15672",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Host != "avatars.rabbitmq" || got.Port != "15672" || got.User != "guest" || got.Password != "guest" {
		t.Fatalf("unexpected config: %+v", got)
	}
}
