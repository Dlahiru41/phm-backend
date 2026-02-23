package store

import (
	"context"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VaccineStore struct {
	pool *pgxpool.Pool
}

func NewVaccineStore(pool *pgxpool.Pool) *VaccineStore { return &VaccineStore{pool: pool} }

func (s *VaccineStore) ListActive(ctx context.Context) ([]models.Vaccine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, manufacturer, dosage_info, recommended_age, interval_days, description, is_active
		FROM vaccines WHERE is_active = true ORDER BY recommended_age, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Vaccine
	for rows.Next() {
		var v models.Vaccine
		err := rows.Scan(&v.VaccineId, &v.Name, &v.Manufacturer, &v.DosageInfo, &v.RecommendedAge, &v.IntervalDays, &v.Description, &v.IsActive)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (s *VaccineStore) GetByID(ctx context.Context, vaccineID string) (*models.Vaccine, error) {
	var v models.Vaccine
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, manufacturer, dosage_info, recommended_age, interval_days, description, is_active
		FROM vaccines WHERE id = $1
	`, vaccineID).Scan(&v.VaccineId, &v.Name, &v.Manufacturer, &v.DosageInfo, &v.RecommendedAge, &v.IntervalDays, &v.Description, &v.IsActive)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
