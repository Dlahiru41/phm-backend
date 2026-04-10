package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MOHTempPasswordStore struct {
	pool *pgxpool.Pool
}

func NewMOHTempPasswordStore(pool *pgxpool.Pool) *MOHTempPasswordStore {
	return &MOHTempPasswordStore{pool: pool}
}

// MOHTempPassword represents a temporary password for MOH account creation
type MOHTempPassword struct {
	ID           string
	EmployeeID   string
	Email        string
	NIC          string
	Name         string
	PhoneNumber  string
	AssignedArea string
	AdminID      string
	TempPassword string
	UsedAt       *time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// Create creates a new MOH temp password record
func (s *MOHTempPasswordStore) Create(ctx context.Context, id, employeeID, email, nic, name, phone, assignedArea, adminID, tempPassword string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO moh_account_temp_passwords (id, employee_id, email, nic, name, phone_number, assigned_area, admin_id, temp_password, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, employeeID, email, nic, name, phone, assignedArea, adminID, tempPassword, expiresAt)
	return err
}

// GetByEmail retrieves a temp password by email
func (s *MOHTempPasswordStore) GetByEmail(ctx context.Context, email string) (*MOHTempPassword, error) {
	var tp MOHTempPassword
	var usedAt *time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT id, employee_id, email, nic, name, phone_number, assigned_area, admin_id, temp_password, 
		       used_at, expires_at, created_at
		FROM moh_account_temp_passwords
		WHERE email = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, email).Scan(
		&tp.ID, &tp.EmployeeID, &tp.Email, &tp.NIC, &tp.Name,
		&tp.PhoneNumber, &tp.AssignedArea, &tp.AdminID, &tp.TempPassword,
		&usedAt, &tp.ExpiresAt, &tp.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	tp.UsedAt = usedAt
	return &tp, nil
}

// GetByID retrieves a temp password by ID
func (s *MOHTempPasswordStore) GetByID(ctx context.Context, id string) (*MOHTempPassword, error) {
	var tp MOHTempPassword
	var usedAt *time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT id, employee_id, email, nic, name, phone_number, assigned_area, admin_id, temp_password, 
		       used_at, expires_at, created_at
		FROM moh_account_temp_passwords
		WHERE id = $1
	`, id).Scan(
		&tp.ID, &tp.EmployeeID, &tp.Email, &tp.NIC, &tp.Name,
		&tp.PhoneNumber, &tp.AssignedArea, &tp.AdminID, &tp.TempPassword,
		&usedAt, &tp.ExpiresAt, &tp.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	tp.UsedAt = usedAt
	return &tp, nil
}

// MarkAsUsed marks the temp password as used
func (s *MOHTempPasswordStore) MarkAsUsed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE moh_account_temp_passwords
		SET used_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

// DeleteExpired deletes expired temporary password records
func (s *MOHTempPasswordStore) DeleteExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM moh_account_temp_passwords
		WHERE expires_at <= NOW() AND used_at IS NULL
	`)
	return err
}

// GetByAdminID retrieves all temp password records created by an admin
func (s *MOHTempPasswordStore) GetByAdminID(ctx context.Context, adminID string) ([]MOHTempPassword, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, employee_id, email, nic, name, phone_number, assigned_area, admin_id, temp_password, 
		       used_at, expires_at, created_at
		FROM moh_account_temp_passwords
		WHERE admin_id = $1
		ORDER BY created_at DESC
	`, adminID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tps []MOHTempPassword
	for rows.Next() {
		var tp MOHTempPassword
		var usedAt *time.Time

		err := rows.Scan(
			&tp.ID, &tp.EmployeeID, &tp.Email, &tp.NIC, &tp.Name,
			&tp.PhoneNumber, &tp.AssignedArea, &tp.AdminID, &tp.TempPassword,
			&usedAt, &tp.ExpiresAt, &tp.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		tp.UsedAt = usedAt
		tps = append(tps, tp)
	}

	return tps, rows.Err()
}
