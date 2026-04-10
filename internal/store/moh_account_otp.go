package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MOHAccountOTPStore struct {
	pool *pgxpool.Pool
}

func NewMOHAccountOTPStore(pool *pgxpool.Pool) *MOHAccountOTPStore {
	return &MOHAccountOTPStore{pool: pool}
}

// MOHAccountOTP represents a one-time password for MOH account creation
type MOHAccountOTP struct {
	ID           string
	AdminID      string
	EmployeeID   string
	Email        string
	NIC          string
	Name         string
	PhoneNumber  string
	AssignedArea string
	OTPHash      string
	AttemptCount int
	MaxAttempts  int
	ConsumedAt   *time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// Create creates a new MOH account OTP record
func (s *MOHAccountOTPStore) Create(ctx context.Context, id, adminID, employeeID, email, nic, name, phone, assignedArea, otpHash string, expiresAt time.Time, maxAttempts int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO moh_account_otps (id, admin_id, employee_id, email, nic, name, phone_number, assigned_area, otp_hash, expires_at, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, adminID, employeeID, email, nic, name, phone, assignedArea, otpHash, expiresAt, maxAttempts)
	return err
}

// GetLatestActive returns the latest active (unconsumed) OTP for the given admin and email
func (s *MOHAccountOTPStore) GetLatestActive(ctx context.Context, adminID, email string) (*MOHAccountOTP, error) {
	var otp MOHAccountOTP
	var consumedAt *time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT id, admin_id, employee_id, email, nic, name, phone_number, assigned_area, otp_hash, 
		       attempt_count, max_attempts, consumed_at, expires_at, created_at
		FROM moh_account_otps
		WHERE admin_id = $1 AND email = $2 AND consumed_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, adminID, email).Scan(
		&otp.ID, &otp.AdminID, &otp.EmployeeID, &otp.Email, &otp.NIC, &otp.Name,
		&otp.PhoneNumber, &otp.AssignedArea, &otp.OTPHash, &otp.AttemptCount,
		&otp.MaxAttempts, &consumedAt, &otp.ExpiresAt, &otp.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	otp.ConsumedAt = consumedAt
	return &otp, nil
}

// GetByID retrieves an OTP by ID
func (s *MOHAccountOTPStore) GetByID(ctx context.Context, id string) (*MOHAccountOTP, error) {
	var otp MOHAccountOTP
	var consumedAt *time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT id, admin_id, employee_id, email, nic, name, phone_number, assigned_area, otp_hash, 
		       attempt_count, max_attempts, consumed_at, expires_at, created_at
		FROM moh_account_otps
		WHERE id = $1
	`, id).Scan(
		&otp.ID, &otp.AdminID, &otp.EmployeeID, &otp.Email, &otp.NIC, &otp.Name,
		&otp.PhoneNumber, &otp.AssignedArea, &otp.OTPHash, &otp.AttemptCount,
		&otp.MaxAttempts, &consumedAt, &otp.ExpiresAt, &otp.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	otp.ConsumedAt = consumedAt
	return &otp, nil
}

// IncrementAttempt increments the attempt count for an OTP
func (s *MOHAccountOTPStore) IncrementAttempt(ctx context.Context, id string) (int, error) {
	var newAttemptCount int
	err := s.pool.QueryRow(ctx, `
		UPDATE moh_account_otps
		SET attempt_count = attempt_count + 1
		WHERE id = $1 AND consumed_at IS NULL AND expires_at > NOW()
		RETURNING attempt_count
	`, id).Scan(&newAttemptCount)
	return newAttemptCount, err
}

// ConsumeValid verifies the OTP and marks it as consumed if correct
func (s *MOHAccountOTPStore) ConsumeValid(ctx context.Context, id, otpHash string) (bool, error) {
	var result int
	err := s.pool.QueryRow(ctx, `
		UPDATE moh_account_otps
		SET consumed_at = NOW()
		WHERE id = $1 AND otp_hash = $2 AND consumed_at IS NULL AND expires_at > NOW()
		RETURNING 1
	`, id, otpHash).Scan(&result)

	if err != nil {
		return false, err
	}
	return true, nil
}

// InvalidateActive marks all active OTPs for an email as expired
func (s *MOHAccountOTPStore) InvalidateActive(ctx context.Context, email string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE moh_account_otps
		SET expires_at = NOW()
		WHERE email = $1 AND consumed_at IS NULL AND expires_at > NOW()
	`, email)
	return err
}

// CountAdminOTPsCreatedToday returns the count of OTPs created by admin today
func (s *MOHAccountOTPStore) CountAdminOTPsCreatedToday(ctx context.Context, adminID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM moh_account_otps
		WHERE admin_id = $1 AND DATE(created_at) = CURRENT_DATE
	`, adminID).Scan(&count)
	return count, err
}
