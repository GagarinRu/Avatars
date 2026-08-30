package models

import "time"

type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusReady      ProcessingStatus = "ready"
	StatusFailed     ProcessingStatus = "failed"
)

func ValidStatus(s ProcessingStatus) bool {
	switch s {
	case StatusPending, StatusProcessing, StatusReady, StatusFailed:
		return true
	default:
		return false
	}
}

type Avatar struct {
	UserID       string           `json:"user_id"`
	Status       ProcessingStatus `json:"status"`
	StagingKey   string           `json:"staging_key,omitempty"`
	OriginalKey  string           `json:"original_key,omitempty"`
	ThumbnailKey string           `json:"thumbnail_key,omitempty"`
	ContentType  string           `json:"content_type,omitempty"`
	SizeBytes    int64            `json:"size_bytes,omitempty"`
	Width        int              `json:"width,omitempty"`
	Height       int              `json:"height,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	MessageID    string           `json:"message_id,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}
