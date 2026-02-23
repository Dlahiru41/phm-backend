package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore { return &NotificationStore{pool: pool} }

func (s *NotificationStore) Create(ctx context.Context, id, recipientID, ntype, message string, relatedChildID *string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO notifications (id, recipient_id, type, message, related_child_id) VALUES ($1, $2, $3, $4, $5)`,
		id, recipientID, ntype, message, relatedChildID)
	return err
}

func (s *NotificationStore) List(ctx context.Context, recipientID string, unreadOnly bool, page, limit int) (total, unreadCount int, list []models.Notification, err error) {
	args := []interface{}{recipientID}
	where := `WHERE recipient_id = $1`
	if unreadOnly {
		where += ` AND is_read = false`
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications `+where, args...).Scan(&total)
	if err != nil {
		return 0, 0, nil, err
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = false`, recipientID).Scan(&unreadCount)
	if err != nil {
		return 0, 0, nil, err
	}
	args = append(args, limit, (page-1)*limit)
	n := len(args)
	rows, err := s.pool.Query(ctx, `
		SELECT id, recipient_id, type, message, related_child_id, sent_date, is_read FROM notifications
		`+where+` ORDER BY sent_date DESC LIMIT $`+fmt.Sprint(n-1)+` OFFSET $`+fmt.Sprint(n), args...)
	if err != nil {
		return 0, 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n models.Notification
		var rel *string
		err := rows.Scan(&n.NotificationId, &n.RecipientId, &n.Type, &n.Message, &rel, &n.SentDate, &n.IsRead)
		if err != nil {
			return 0, 0, nil, err
		}
		n.RelatedChildId = rel
		list = append(list, n)
	}
	return total, unreadCount, list, rows.Err()
}

func (s *NotificationStore) MarkRead(ctx context.Context, notificationID, recipientID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET is_read = true WHERE id = $1 AND recipient_id = $2`, notificationID, recipientID)
	return err
}

func (s *NotificationStore) MarkAllRead(ctx context.Context, recipientID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET is_read = true WHERE recipient_id = $1`, recipientID)
	return err
}
