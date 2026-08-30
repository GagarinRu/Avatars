package queue

type AvatarProcessMessage struct {
	MessageID  string `json:"message_id"`
	UserID     string `json:"user_id"`
	StagingKey string `json:"staging_key"`
}
