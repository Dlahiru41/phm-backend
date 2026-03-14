package store

import (
	"context"
	"time"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChildLinkOTPStore struct {
	pool *pgxpool.Pool
}

func NewChildLinkOTPStore(pool *pgxpool.Pool) *ChildLinkOTPStore {
	return &ChildLinkOTPStore{pool: pool}
}

func (s *ChildLinkOTPStore) Create(ctx context.Context, id, childID, parentID, otpHash string, expiresAt time.Time, maxAttempts int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO child_link_otps (id, child_id, parent_id, otp_hash, expires_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, childID, parentID, otpHash, expiresAt, maxAttempts)
	return err
}

func (s *ChildLinkOTPStore) InvalidateActive(ctx context.Context, childID, parentID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE child_link_otps
		SET consumed_at = NOW()
		WHERE child_id = $1
		  AND parent_id = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
	`, childID, parentID)
	return err
}

func (s *ChildLinkOTPStore) GetLatestActive(ctx context.Context, childID, parentID string) (*models.ChildLinkOTP, error) {
	var o models.ChildLinkOTP
	err := s.pool.QueryRow(ctx, `
		SELECT id, child_id, parent_id, otp_hash, expires_at, attempt_count, max_attempts, created_at, consumed_at
		FROM child_link_otps
		WHERE child_id = $1
		  AND parent_id = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, childID, parentID).Scan(&o.ID, &o.ChildID, &o.ParentID, &o.OTPHash, &o.ExpiresAt, &o.AttemptCount, &o.MaxAttempts, &o.CreatedAt, &o.ConsumedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (s *ChildLinkOTPStore) ConsumeValid(ctx context.Context, childID, parentID, otpHash string) (bool, error) {
	var otpID string
	err := s.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT id
			FROM child_link_otps
			WHERE child_id = $1
			  AND parent_id = $2
			  AND otp_hash = $3
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			  AND attempt_count < max_attempts
			ORDER BY created_at DESC
			LIMIT 1
		)
		UPDATE child_link_otps o
		SET consumed_at = NOW()
		FROM candidate c
		WHERE o.id = c.id
		RETURNING o.id
	`, childID, parentID, otpHash).Scan(&otpID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return otpID != "", nil
}

func (s *ChildLinkOTPStore) IncrementAttempt(ctx context.Context, childID, parentID string) (*models.ChildLinkOTP, error) {
	var o models.ChildLinkOTP
	err := s.pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT id
			FROM child_link_otps
			WHERE child_id = $1
			  AND parent_id = $2
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			ORDER BY created_at DESC
			LIMIT 1
		)
		UPDATE child_link_otps o
		SET attempt_count = attempt_count + 1
		FROM latest l
		WHERE o.id = l.id
		RETURNING o.id, o.child_id, o.parent_id, o.otp_hash, o.expires_at, o.attempt_count, o.max_attempts, o.created_at, o.consumed_at
	`, childID, parentID).Scan(&o.ID, &o.ChildID, &o.ParentID, &o.OTPHash, &o.ExpiresAt, &o.AttemptCount, &o.MaxAttempts, &o.CreatedAt, &o.ConsumedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}
