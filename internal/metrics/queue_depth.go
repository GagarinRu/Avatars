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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var QueueDepth = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "avatars_queue_depth",
		Help: "Number of messages waiting in the processing queue",
	},
)

var managementClient = &http.Client{Timeout: 10 * time.Second}

// RabbitMQManagement holds connection details for the RabbitMQ management API.
type RabbitMQManagement struct {
	Host     string
	Port     string
	User     string
	Password string
}

func RabbitMQManagementFromURL(rabbitURL, mgmtPort, user, password string) (RabbitMQManagement, error) {
	parsed, err := url.Parse(rabbitURL)
	if err != nil {
		return RabbitMQManagement{}, err
	}
	if parsed.Scheme != "amqp" && parsed.Scheme != "amqps" {
		return RabbitMQManagement{}, fmt.Errorf("unsupported rabbitmq scheme: %s", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return RabbitMQManagement{}, fmt.Errorf("rabbitmq host is empty")
	}
	if mgmtPort == "" {
		mgmtPort = "15672"
	}
	if user == "" {
		user = parsed.User.Username()
	}
	if password == "" {
		password, _ = parsed.User.Password()
	}
	if user == "" {
		user = "guest"
		password = "guest"
	}
	return RabbitMQManagement{
		Host:     host,
		Port:     mgmtPort,
		User:     user,
		Password: password,
	}, nil
}

func StartQueueDepthPoller(ctx context.Context, mgmt RabbitMQManagement, queueName string, interval time.Duration) {
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
				depth, err := fetchQueueDepth(mgmt, queueName)
				if err != nil {
					slog.Debug("queue depth poll failed", "error", err)
					continue
				}
				QueueDepth.Set(float64(depth))
			}
		}
	}()
}

func fetchQueueDepth(mgmt RabbitMQManagement, queueName string) (int, error) {
	baseURL, err := rabbitManagementBaseURL(mgmt)
	if err != nil {
		return 0, err
	}
	endpoint := fmt.Sprintf("%s/api/queues/%s/%s", baseURL, url.PathEscape("/"), url.PathEscape(queueName))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(mgmt.User, mgmt.Password)
	resp, err := managementClient.Do(req)
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

func rabbitManagementBaseURL(mgmt RabbitMQManagement) (string, error) {
	if mgmt.Host == "" {
		return "", fmt.Errorf("rabbitmq host is empty")
	}
	port := mgmt.Port
	if port == "" {
		port = "15672"
	}
	return fmt.Sprintf("http://%s:%s", mgmt.Host, port), nil
}
