package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus"
)

var QueueDepth = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "avatars_queue_depth",
		Help: "Number of messages waiting in the processing queue",
	},
)

func StartQueueDepthPoller(ctx context.Context, rabbitURL, queueName string, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				depth, err := fetchQueueDepth(rabbitURL, queueName)
				if err != nil {
					slog.Debug("queue depth poll failed", "error", err)
					continue
				}
				QueueDepth.Set(float64(depth))
			}
		}
	}()
}

func fetchQueueDepth(rabbitURL, queueName string) (int, error) {
	managementURL, err := rabbitManagementURL(rabbitURL)
	if err != nil {
		return 0, err
	}
	endpoint := fmt.Sprintf("%s/api/queues/%s/%s", managementURL, url.PathEscape("/"), url.PathEscape(queueName))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("management api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Messages int `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.Messages, nil
}

func RabbitManagementURLForTest(rabbitURL string) (string, error) {
	return rabbitManagementURL(rabbitURL)
}

func rabbitManagementURL(rabbitURL string) (string, error) {
	parsed, err := url.Parse(rabbitURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "amqp" && parsed.Scheme != "amqps" {
		return "", fmt.Errorf("unsupported rabbitmq scheme: %s", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("rabbitmq host is empty")
	}
	port := "15672"
	if override := strings.TrimSpace(parsed.Query().Get("management_port")); override != "" {
		port = override
	}
	user := parsed.User.Username()
	password, _ := parsed.User.Password()
	if user == "" {
		user = "guest"
		password = "guest"
	}
	return fmt.Sprintf("http://%s:%s@%s:%s", user, password, host, port), nil
}
