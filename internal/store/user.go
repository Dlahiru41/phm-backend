package store

import (
	"context"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool: pool} }

func (s *UserStore) GetByID(ctx context.Context, id string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	var areaCode *string
	var notif []byte
	var employeeId *string
	var assignedArea *string
	var createdByMoh *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, COALESCE(phone_number, ''), 
		       COALESCE(address, ''), language_preference, COALESCE(notification_settings::text,'{}'), 
		       area_code, created_at, updated_at, first_login, employee_id, assigned_area, created_by_moh
		FROM users WHERE id = $1
	`, id).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &notif, &areaCode, &u.CreatedAt, &u.UpdatedAt, &u.FirstLogin,
		&employeeId, &assignedArea, &createdByMoh)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
	u.EmployeeId = employeeId
	u.AssignedArea = assignedArea
	u.CreatedByMoh = createdByMoh
	_ = notif
	return &u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	var areaCode *string
	var notifJSON []byte
	var employeeId *string
	var assignedArea *string
	var createdByMoh *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, COALESCE(phone_number, ''), 
		       COALESCE(address, ''), language_preference, COALESCE(notification_settings::text,'{}'), 
		       area_code, created_at, updated_at, first_login, employee_id, assigned_area, created_by_moh
		FROM users WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &notifJSON, &areaCode, &u.CreatedAt, &u.UpdatedAt, &u.FirstLogin,
		&employeeId, &assignedArea, &createdByMoh)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
	u.EmployeeId = employeeId
	u.AssignedArea = assignedArea
	u.CreatedByMoh = createdByMoh
	_ = notifJSON
	return &u, nil
}

func (s *UserStore) GetByNIC(ctx context.Context, nic string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	var areaCode *string
	var employeeId *string
	var assignedArea *string
	var createdByMoh *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, COALESCE(phone_number, ''), 
		       COALESCE(address, ''), language_preference, area_code, created_at, updated_at,
		       first_login, employee_id, assigned_area, created_by_moh
		FROM users WHERE nic = $1
	`, nic).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &areaCode, &u.CreatedAt, &u.UpdatedAt, &u.FirstLogin,
		&employeeId, &assignedArea, &createdByMoh)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
	u.EmployeeId = employeeId
	u.AssignedArea = assignedArea
	u.CreatedByMoh = createdByMoh
	return &u, nil
}

func (s *UserStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT 1 FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&n)
	return n == 1, err
}

func (s *UserStore) ExistsByNIC(ctx context.Context, nic string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT 1 FROM users WHERE nic = $1`, nic).Scan(&n)
	return n == 1, err
}

func (s *UserStore) ExistsByPhoneForOther(ctx context.Context, phone, excludeUserID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1 AND id <> $2)`, phone, excludeUserID).Scan(&exists)
	return exists, err
}

func (s *UserStore) Create(ctx context.Context, id, email, nic, passwordHash, role, name, phone, address string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, nic, password_hash, role, name, phone_number, address, language_preference)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'en')
	`, id, email, nic, passwordHash, role, name, phone, address)
	return err
}

func (s *UserStore) UpdateProfile(ctx context.Context, id string, name, phone, address, lang *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET name = COALESCE($2, name), phone_number = COALESCE($3, phone_number),
		                 address = COALESCE($4, address), language_preference = COALESCE($5, language_preference)
		WHERE id = $1
	`, id, name, phone, address, lang)
	return err
}

func (s *UserStore) UpdateSettings(ctx context.Context, id string, lang *string, notif map[string]interface{}) error {
	if lang != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE users SET language_preference = $2 WHERE id = $1`, id, *lang)
	}
	if notif != nil {
		_, err := s.pool.Exec(ctx, `UPDATE users SET notification_settings = $2 WHERE id = $1`, id, notif)
		return err
	}
	return nil
}

func (s *UserStore) SavePasswordResetToken(ctx context.Context, token, userID string, expiresAt interface{}) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO password_reset_tokens (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt)
	return err
}

func (s *UserStore) ConsumeResetToken(ctx context.Context, token string) (userID string, err error) {
	err = s.pool.QueryRow(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, token).Scan(&userID)
	return userID, err
}

func (s *UserStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	return err
}

func (s *UserStore) CreatePHM(ctx context.Context, id, employeeId, email, nic, passwordHash, name, phone, assignedArea, createdByMohID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, employee_id, email, nic, password_hash, role, name, phone_number, assigned_area, created_by_moh, first_login, language_preference)
		VALUES ($1, $2, $3, $4, $5, 'phm', $6, $7, $8, $9, true, 'en')
	`, id, employeeId, email, nic, passwordHash, name, phone, assignedArea, createdByMohID)
	return err
}

func (s *UserStore) CompleteFirstLogin(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET first_login = false, updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (s *UserStore) UpdatePhoneNumber(ctx context.Context, userID, phone string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET phone_number = $2, updated_at = NOW() WHERE id = $1`, userID, phone)
	return err
}

// CreateMOH creates a new MOH account with admin as creator
func (s *UserStore) CreateMOH(ctx context.Context, id, employeeId, email, nic, passwordHash, name, phone, assignedArea, createdByAdminID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, employee_id, email, nic, password_hash, role, name, phone_number, assigned_area, created_by_moh, first_login, language_preference)
		VALUES ($1, $2, $3, $4, $5, 'moh', $6, $7, $8, $9, true, 'en')
	`, id, employeeId, email, nic, passwordHash, name, phone, assignedArea, createdByAdminID)
	return err
}

// ListMOHUsers returns all MOH users with non-sensitive profile details.
func (s *UserStore) ListMOHUsers(ctx context.Context) ([]models.MOHUserSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(employee_id, ''), name, email, COALESCE(phone_number, ''),
		       COALESCE(assigned_area, ''), first_login, created_at
		FROM users
		WHERE role = 'moh'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.MOHUserSummary, 0)
	for rows.Next() {
		var item models.MOHUserSummary
		if err := rows.Scan(
			&item.UserId,
			&item.EmployeeId,
			&item.Name,
			&item.Email,
			&item.PhoneNumber,
			&item.AssignedArea,
			&item.FirstLogin,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// IsAdmin checks if a user has admin role
func (s *UserStore) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
	if err != nil {
		return false, err
	}
	return role == "admin", nil
}

// CountAdminUsers returns the count of admin users in the system
func (s *UserStore) CountAdminUsers(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count)
	return count, err
}
