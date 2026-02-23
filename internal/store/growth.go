package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GrowthRecordStore struct {
	pool *pgxpool.Pool
}

func NewGrowthRecordStore(pool *pgxpool.Pool) *GrowthRecordStore { return &GrowthRecordStore{pool: pool} }

func (s *GrowthRecordStore) Create(ctx context.Context, id, childID, recordedDate, recordedBy, notes string,
	height, weight, headCirc *float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO growth_records (id, child_id, recorded_date, height, weight, head_circumference, recorded_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, childID, recordedDate, height, weight, headCirc, recordedBy, notes)
	return err
}

func (s *GrowthRecordStore) ByChildID(ctx context.Context, childID, startDate, endDate string) ([]models.GrowthRecord, error) {
	q := `SELECT id, child_id, recorded_date::text, height, weight, head_circumference, recorded_by, notes, created_at FROM growth_records WHERE child_id = $1`
	args := []interface{}{childID}
	if startDate != "" {
		q += ` AND recorded_date >= $2`
		args = append(args, startDate)
	}
	if endDate != "" {
		q += fmt.Sprintf(` AND recorded_date <= $%d`, len(args)+1)
		args = append(args, endDate)
	}
	q += ` ORDER BY recorded_date DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.GrowthRecord
	for rows.Next() {
		var r models.GrowthRecord
		err := rows.Scan(&r.RecordId, &r.ChildId, &r.RecordedDate, &r.Height, &r.Weight, &r.HeadCircumference, &r.RecordedBy, &r.Notes, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
