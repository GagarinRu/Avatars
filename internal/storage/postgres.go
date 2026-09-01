package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/GagarinRu/avatars/internal/logger"
	"github.com/GagarinRu/avatars/internal/models"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	logger.Log.Info("Connected to PostgreSQL")
	return &PostgresStorage{db: db}, nil
}

func (ps *PostgresStorage) Close() error {
	return ps.db.Close()
}

func (ps *PostgresStorage) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}

func (ps *PostgresStorage) UpsertAvatar(ctx context.Context, avatar *models.Avatar) error {
	now := time.Now().UTC()
	if avatar.CreatedAt.IsZero() {
		avatar.CreatedAt = now
	}
	avatar.UpdatedAt = now
	_, err := ps.db.ExecContext(ctx, `
		INSERT INTO avatars (
			user_id, status, staging_key, original_key, thumbnail_key,
			content_type, size_bytes, width, height, error_message, message_id,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			staging_key = EXCLUDED.staging_key,
			original_key = EXCLUDED.original_key,
			thumbnail_key = EXCLUDED.thumbnail_key,
			content_type = EXCLUDED.content_type,
			size_bytes = EXCLUDED.size_bytes,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			error_message = EXCLUDED.error_message,
			message_id = EXCLUDED.message_id,
			updated_at = EXCLUDED.updated_at
	`, avatar.UserID, avatar.Status, avatar.StagingKey, avatar.OriginalKey, avatar.ThumbnailKey,
		avatar.ContentType, avatar.SizeBytes, avatar.Width, avatar.Height, nullString(avatar.ErrorMessage),
		avatar.MessageID, avatar.CreatedAt, avatar.UpdatedAt)
	return err
}

func (ps *PostgresStorage) GetAvatar(ctx context.Context, userID string) (*models.Avatar, error) {
	row := ps.db.QueryRowContext(ctx, `
		SELECT user_id, status, staging_key, original_key, thumbnail_key,
			content_type, size_bytes, width, height, COALESCE(error_message,''), message_id,
			created_at, updated_at
		FROM avatars WHERE user_id = $1
	`, userID)
	return scanAvatar(row)
}

func (ps *PostgresStorage) DeleteAvatar(ctx context.Context, userID string) error {
	res, err := ps.db.ExecContext(ctx, `DELETE FROM avatars WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (ps *PostgresStorage) UpdateAvatarStatus(ctx context.Context, userID string, status models.ProcessingStatus, errMsg string) error {
	_, err := ps.db.ExecContext(ctx, `
		UPDATE avatars SET status = $2, error_message = $3, updated_at = $4 WHERE user_id = $1
	`, userID, status, nullString(errMsg), time.Now().UTC())
	return err
}

func (ps *PostgresStorage) MarkAvatarReady(ctx context.Context, userID, originalKey, thumbKey, contentType string, size int64, width, height int) error {
	_, err := ps.db.ExecContext(ctx, `
		UPDATE avatars SET
			status = $2, original_key = $3, thumbnail_key = $4, content_type = $5,
			size_bytes = $6, width = $7, height = $8, staging_key = '', error_message = NULL,
			updated_at = $9
		WHERE user_id = $1
	`, userID, models.StatusReady, originalKey, thumbKey, contentType, size, width, height, time.Now().UTC())
	return err
}

func (ps *PostgresStorage) IsMessageProcessed(ctx context.Context, messageID string) (bool, error) {
	var exists bool
	err := ps.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM processed_messages WHERE message_id = $1)
	`, messageID).Scan(&exists)
	return exists, err
}

func (ps *PostgresStorage) MarkMessageProcessed(ctx context.Context, messageID string) error {
	_, err := ps.db.ExecContext(ctx, `
		INSERT INTO processed_messages (message_id, processed_at) VALUES ($1, $2)
		ON CONFLICT (message_id) DO NOTHING
	`, messageID, time.Now().UTC())
	return err
}

func scanAvatar(row *sql.Row) (*models.Avatar, error) {
	var a models.Avatar
	var errMsg sql.NullString
	err := row.Scan(
		&a.UserID, &a.Status, &a.StagingKey, &a.OriginalKey, &a.ThumbnailKey,
		&a.ContentType, &a.SizeBytes, &a.Width, &a.Height, &errMsg, &a.MessageID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errMsg.Valid {
		a.ErrorMessage = errMsg.String
	}
	return &a, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

var ErrNotFound = errors.New("not found")
