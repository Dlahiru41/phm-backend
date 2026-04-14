package store

import (
	"context"
	"fmt"
	"strings"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClinicStore struct {
	pool *pgxpool.Pool
}

func NewClinicStore(pool *pgxpool.Pool) *ClinicStore { return &ClinicStore{pool: pool} }

func (s *ClinicStore) Create(ctx context.Context, clinic *models.ClinicSchedule) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO clinic_schedules (id, phm_id, clinic_date, gn_division, location, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, clinic.ClinicId, clinic.PhmId, clinic.ClinicDate, clinic.GnDivision, clinic.Location, clinic.Description, clinic.Status, clinic.CreatedAt, clinic.UpdatedAt)
	return err
}

func (s *ClinicStore) GetByID(ctx context.Context, clinicID string) (*models.ClinicSchedule, error) {
	var clinic models.ClinicSchedule
	err := s.pool.QueryRow(ctx, `
		SELECT id, phm_id, clinic_date::text, gn_division, location, COALESCE(description, ''), status, created_at, updated_at
		FROM clinic_schedules
		WHERE id = $1
	`, clinicID).Scan(
		&clinic.ClinicId,
		&clinic.PhmId,
		&clinic.ClinicDate,
		&clinic.GnDivision,
		&clinic.Location,
		&clinic.Description,
		&clinic.Status,
		&clinic.CreatedAt,
		&clinic.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &clinic, nil
}

func (s *ClinicStore) ListByPHM(ctx context.Context, phmID string, fromDate, toDate *string) ([]models.ClinicSchedule, error) {
	args := []interface{}{phmID}
	where := []string{"phm_id = $1"}

	if fromDate != nil && strings.TrimSpace(*fromDate) != "" {
		args = append(args, strings.TrimSpace(*fromDate))
		where = append(where, fmt.Sprintf("clinic_date >= $%d", len(args)))
	}
	if toDate != nil && strings.TrimSpace(*toDate) != "" {
		args = append(args, strings.TrimSpace(*toDate))
		where = append(where, fmt.Sprintf("clinic_date <= $%d", len(args)))
	}

	query := `
		SELECT id, phm_id, clinic_date::text, gn_division, location, COALESCE(description, ''), status, created_at, updated_at
		FROM clinic_schedules
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY clinic_date DESC, created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clinics []models.ClinicSchedule
	for rows.Next() {
		var clinic models.ClinicSchedule
		err := rows.Scan(
			&clinic.ClinicId,
			&clinic.PhmId,
			&clinic.ClinicDate,
			&clinic.GnDivision,
			&clinic.Location,
			&clinic.Description,
			&clinic.Status,
			&clinic.CreatedAt,
			&clinic.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		clinics = append(clinics, clinic)
	}

	return clinics, rows.Err()
}

func (s *ClinicStore) GetDueChildren(ctx context.Context, clinicID string) ([]models.DueChild, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			c.id,
			c.first_name,
			c.last_name,
			c.registration_number,
			c.date_of_birth::text,
			v.name,
			vs.due_date::text,
			c.parent_id,
			p.name,
			p.phone_number,
			vr.dose_number
		FROM clinic_schedules cs
		JOIN children c ON c.gn_division = cs.gn_division
		JOIN vaccination_schedules vs ON vs.child_id = c.id
		JOIN vaccines v ON v.id = vs.vaccine_id
		LEFT JOIN users p ON p.id = c.parent_id
		LEFT JOIN vaccination_records vr ON vr.child_id = c.id AND vr.vaccine_id = vs.vaccine_id
		WHERE cs.id = $1
		  AND vs.status IN ('pending', 'scheduled', 'missed')
		  AND vs.due_date BETWEEN (cs.clinic_date - INTERVAL '7 days') AND cs.clinic_date
		ORDER BY c.id, vs.due_date ASC
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.DueChild
	for rows.Next() {
		var d models.DueChild
		err := rows.Scan(
			&d.ChildId,
			&d.FirstName,
			&d.LastName,
			&d.RegistrationNumber,
			&d.DateOfBirth,
			&d.VaccineName,
			&d.NextDueDate,
			&d.ParentId,
			&d.ParentName,
			&d.ParentPhone,
			&d.DoseNumber,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}

	return list, rows.Err()
}

func (s *ClinicStore) CreateClinicChild(ctx context.Context, clinicChild *models.ClinicChild) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO clinic_children (id, clinic_id, child_id, attended, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, clinicChild.ClinicChildId, clinicChild.ClinicId, clinicChild.ChildId, clinicChild.Attended, clinicChild.CreatedAt, clinicChild.UpdatedAt)
	return err
}

func (s *ClinicStore) GetClinicChildren(ctx context.Context, clinicID string) ([]models.ClinicChild, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, clinic_id, child_id, attended, created_at, updated_at
		FROM clinic_children
		WHERE clinic_id = $1
		ORDER BY created_at ASC
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ClinicChild
	for rows.Next() {
		var c models.ClinicChild
		err := rows.Scan(&c.ClinicChildId, &c.ClinicId, &c.ChildId, &c.Attended, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}

	return list, rows.Err()
}

func (s *ClinicStore) UpdateClinicStatus(ctx context.Context, clinicID, status string) error {
	res, err := s.pool.Exec(ctx, `
		UPDATE clinic_schedules
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, clinicID, status)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *ClinicStore) UpdateClinicChildAttendance(ctx context.Context, clinicID, childID string, attended bool) error {
	res, err := s.pool.Exec(ctx, `
		UPDATE clinic_children
		SET attended = $3, updated_at = NOW()
		WHERE clinic_id = $1 AND child_id = $2
	`, clinicID, childID, attended)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
