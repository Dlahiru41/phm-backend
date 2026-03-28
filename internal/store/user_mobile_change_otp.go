package store

import (
	"context"
	"time"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserMobileChangeOTPStore struct {
	pool *pgxpool.Pool
}

func NewUserMobileChangeOTPStore(pool *pgxpool.Pool) *UserMobileChangeOTPStore {
	return &UserMobileChangeOTPStore{pool: pool}
}

func (s *UserMobileChangeOTPStore) Create(ctx context.Context, id, userID, newPhone, otpHash string, expiresAt time.Time, maxAttempts int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_mobile_change_otps (id, user_id, new_phone_number, otp_hash, expires_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, userID, newPhone, otpHash, expiresAt, maxAttempts)
	return err
}

func (s *UserMobileChangeOTPStore) InvalidateActive(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_mobile_change_otps
		SET consumed_at = NOW()
		WHERE user_id = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
	`, userID)
	return err
}

func (s *UserMobileChangeOTPStore) GetLatestActive(ctx context.Context, userID string) (*models.UserMobileChangeOTP, error) {
	var o models.UserMobileChangeOTP
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, new_phone_number, otp_hash, expires_at, attempt_count, max_attempts, created_at, consumed_at
		FROM user_mobile_change_otps
		WHERE user_id = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&o.ID, &o.UserID, &o.NewPhone, &o.OTPHash, &o.ExpiresAt, &o.AttemptCount, &o.MaxAttempts, &o.CreatedAt, &o.ConsumedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (s *UserMobileChangeOTPStore) ConsumeValid(ctx context.Context, userID, newPhone, otpHash string) (bool, error) {
	var otpID string
	err := s.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT id
			FROM user_mobile_change_otps
			WHERE user_id = $1
			  AND new_phone_number = $2
			  AND otp_hash = $3
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			  AND attempt_count < max_attempts
			ORDER BY created_at DESC
			LIMIT 1
		)
		UPDATE user_mobile_change_otps o
		SET consumed_at = NOW()
		FROM candidate c
		WHERE o.id = c.id
		RETURNING o.id
	`, userID, newPhone, otpHash).Scan(&otpID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return otpID != "", nil
}

func (s *UserMobileChangeOTPStore) IncrementAttempt(ctx context.Context, userID, newPhone string) (*models.UserMobileChangeOTP, error) {
	var o models.UserMobileChangeOTP
	err := s.pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT id
			FROM user_mobile_change_otps
			WHERE user_id = $1
			  AND new_phone_number = $2
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			ORDER BY created_at DESC
			LIMIT 1
		)
		UPDATE user_mobile_change_otps o
		SET attempt_count = attempt_count + 1
		FROM latest l
		WHERE o.id = l.id
		RETURNING o.id, o.user_id, o.new_phone_number, o.otp_hash, o.expires_at, o.attempt_count, o.max_attempts, o.created_at, o.consumed_at
	`, userID, newPhone).Scan(&o.ID, &o.UserID, &o.NewPhone, &o.OTPHash, &o.ExpiresAt, &o.AttemptCount, &o.MaxAttempts, &o.CreatedAt, &o.ConsumedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (s *UserMobileChangeOTPStore) InvalidateActiveByPhone(ctx context.Context, userID, newPhone string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE user_mobile_change_otps
		SET consumed_at = NOW()
		WHERE user_id = $1
		  AND new_phone_number = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
	`, userID, newPhone)
	return err
}
