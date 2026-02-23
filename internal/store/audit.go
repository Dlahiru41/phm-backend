package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore { return &AuditStore{pool: pool} }

func (s *AuditStore) Insert(ctx context.Context, id string, userID *string, userRole, userName, action, entityType, entityID, details, ipAddress string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, user_role, user_name, action, entity_type, entity_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, userID, userRole, userName, action, entityType, entityID, details, ipAddress)
	return err
}

func (s *AuditStore) List(ctx context.Context, userID, userRole, entityType, action, startDate, endDate, search string, page, limit int) (total int, list []models.AuditLog, err error) {
	base := `FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if userID != "" {
		base += fmt.Sprintf(` AND user_id = $%d`, idx)
		args = append(args, userID)
		idx++
	}
	if userRole != "" {
		base += fmt.Sprintf(` AND user_role = $%d`, idx)
		args = append(args, userRole)
		idx++
	}
	if entityType != "" {
		base += fmt.Sprintf(` AND entity_type = $%d`, idx)
		args = append(args, entityType)
		idx++
	}
	if action != "" {
		base += fmt.Sprintf(` AND action = $%d`, idx)
		args = append(args, action)
		idx++
	}
	if startDate != "" {
		base += fmt.Sprintf(` AND timestamp >= $%d`, idx)
		args = append(args, startDate)
		idx++
	}
	if endDate != "" {
		base += fmt.Sprintf(` AND timestamp <= $%d`, idx)
		args = append(args, endDate)
		idx++
	}
	if search != "" {
		base += fmt.Sprintf(` AND (details ILIKE $%d OR user_name ILIKE $%d)`, idx, idx)
		args = append(args, "%"+search+"%")
		idx++
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) `+base, args...).Scan(&total)
	if err != nil {
		return 0, nil, err
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, user_role, user_name, action, entity_type, entity_id, details, timestamp, ip_address
		`+base+` ORDER BY timestamp DESC LIMIT $`+fmt.Sprint(idx)+` OFFSET $`+fmt.Sprint(idx+1), args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a models.AuditLog
		var uid *string
		err := rows.Scan(&a.LogId, &uid, &a.UserRole, &a.UserName, &a.Action, &a.EntityType, &a.EntityId, &a.Details, &a.Timestamp, &a.IpAddress)
		if err != nil {
			return 0, nil, err
		}
		a.UserId = uid
		list = append(list, a)
	}
	return total, list, rows.Err()
}
