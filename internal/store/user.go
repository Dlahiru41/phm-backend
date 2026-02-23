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
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, phone_number, address,
		       language_preference, COALESCE(notification_settings::text,'{}'), area_code, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &notif, &areaCode, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
	_ = notif
	return &u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	var areaCode *string
	var notifJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, phone_number, address,
		       language_preference, notification_settings, area_code, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &notifJSON, &areaCode, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
	_ = notifJSON
	return &u, nil
}

func (s *UserStore) GetByNIC(ctx context.Context, nic string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	var areaCode *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, nic, password_hash, role, name, phone_number, address,
		       language_preference, area_code, created_at, updated_at
		FROM users WHERE nic = $1
	`, nic).Scan(&u.UserId, &u.Email, &u.NIC, &u.PasswordHash, &u.Role, &u.Name, &u.PhoneNumber, &u.Address,
		&u.LanguagePreference, &areaCode, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.AreaCode = areaCode
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
