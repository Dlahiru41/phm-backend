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
		WITH clinic_scope AS (
			SELECT id, clinic_date, lower(trim(gn_division)) AS gn_key
			FROM clinic_schedules
			WHERE id = $1
		),
		latest_records AS (
			SELECT
				vr.child_id,
				vr.vaccine_id,
				vr.dose_number,
				vr.next_due_date,
				vr.status,
				ROW_NUMBER() OVER (
					PARTITION BY vr.child_id, vr.vaccine_id
					ORDER BY vr.administered_date DESC, vr.created_at DESC
				) AS rn
			FROM vaccination_records vr
			WHERE vr.next_due_date IS NOT NULL
		),
		due_from_schedules AS (
			SELECT
				c.id AS child_id,
				c.first_name,
				c.last_name,
				c.registration_number,
				c.date_of_birth,
				v.name AS vaccine_name,
				vs.due_date AS next_due_date,
				c.parent_id,
				p.name AS parent_name,
				p.phone_number,
				lr.dose_number,
				1 AS source_rank
			FROM clinic_scope cs
			JOIN children c ON lower(trim(c.gn_division)) = cs.gn_key
			JOIN vaccination_schedules vs ON vs.child_id = c.id
			JOIN vaccines v ON v.id = vs.vaccine_id
			LEFT JOIN users p ON p.id = c.parent_id
			LEFT JOIN latest_records lr ON lr.child_id = c.id AND lr.vaccine_id = vs.vaccine_id AND lr.rn = 1
			WHERE vs.status IN ('pending', 'scheduled', 'missed')
			  AND vs.due_date <= cs.clinic_date
		),
		due_from_records AS (
			SELECT
				c.id AS child_id,
				c.first_name,
				c.last_name,
				c.registration_number,
				c.date_of_birth,
				v.name AS vaccine_name,
				lr.next_due_date,
				c.parent_id,
				p.name AS parent_name,
				p.phone_number,
				lr.dose_number,
				2 AS source_rank
			FROM clinic_scope cs
			JOIN children c ON lower(trim(c.gn_division)) = cs.gn_key
			JOIN latest_records lr ON lr.child_id = c.id AND lr.rn = 1
			JOIN vaccines v ON v.id = lr.vaccine_id
			LEFT JOIN users p ON p.id = c.parent_id
			WHERE lr.status <> 'cancelled'
			  AND lr.next_due_date <= cs.clinic_date
		),
		combined_due AS (
			SELECT * FROM due_from_schedules
			UNION ALL
			SELECT * FROM due_from_records
		),
		dedup_due AS (
			SELECT DISTINCT ON (child_id, vaccine_name, next_due_date)
				child_id,
				first_name,
				last_name,
				registration_number,
				date_of_birth,
				vaccine_name,
				next_due_date,
				parent_id,
				parent_name,
				phone_number,
				dose_number
			FROM combined_due
			ORDER BY child_id, vaccine_name, next_due_date, source_rank
		)
		SELECT
			child_id,
			first_name,
			last_name,
			registration_number,
			date_of_birth::text,
			vaccine_name,
			next_due_date::text,
			parent_id,
			parent_name,
			phone_number,
			dose_number
		FROM dedup_due
		ORDER BY next_due_date ASC, child_id ASC
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

func (s *ClinicStore) ListParentDueVaccinations(ctx context.Context, parentID string) ([]models.ParentDueVaccination, error) {
	rows, err := s.pool.Query(ctx, `
		WITH parent_clinics AS (
			SELECT
				cs.id AS clinic_id,
				cs.clinic_date,
				cs.location,
				c.id AS child_id,
				TRIM(CONCAT(c.first_name, ' ', c.last_name)) AS child_name,
				c.registration_number
			FROM clinic_children cc
			JOIN clinic_schedules cs ON cs.id = cc.clinic_id
			JOIN children c ON c.id = cc.child_id
			WHERE c.parent_id = $1
			  AND cs.status = 'scheduled'
			  AND cs.clinic_date >= CURRENT_DATE
		),
		latest_records AS (
			SELECT
				vr.child_id,
				vr.vaccine_id,
				vr.next_due_date,
				vr.status,
				ROW_NUMBER() OVER (
					PARTITION BY vr.child_id, vr.vaccine_id
					ORDER BY vr.administered_date DESC, vr.created_at DESC
				) AS rn
			FROM vaccination_records vr
			WHERE vr.next_due_date IS NOT NULL
		),
		due_from_schedules AS (
			SELECT
				pc.clinic_id,
				pc.clinic_date,
				pc.location,
				pc.child_id,
				pc.child_name,
				pc.registration_number,
				v.name AS vaccine_name,
				vs.due_date AS next_due_date,
				1 AS source_rank
			FROM parent_clinics pc
			JOIN vaccination_schedules vs ON vs.child_id = pc.child_id
			JOIN vaccines v ON v.id = vs.vaccine_id
			LEFT JOIN latest_records lr ON lr.child_id = pc.child_id AND lr.vaccine_id = vs.vaccine_id AND lr.rn = 1
			WHERE vs.status IN ('pending', 'scheduled', 'missed')
			  AND vs.due_date <= pc.clinic_date
		),
		due_from_records AS (
			SELECT
				pc.clinic_id,
				pc.clinic_date,
				pc.location,
				pc.child_id,
				pc.child_name,
				pc.registration_number,
				v.name AS vaccine_name,
				lr.next_due_date,
				2 AS source_rank
			FROM parent_clinics pc
			JOIN latest_records lr ON lr.child_id = pc.child_id AND lr.rn = 1
			JOIN vaccines v ON v.id = lr.vaccine_id
			WHERE lr.status <> 'cancelled'
			  AND lr.next_due_date <= pc.clinic_date
		),
		combined_due AS (
			SELECT * FROM due_from_schedules
			UNION ALL
			SELECT * FROM due_from_records
		),
		dedup_due AS (
			SELECT DISTINCT ON (clinic_id, child_id, vaccine_name, next_due_date)
				clinic_id,
				clinic_date,
				location,
				child_id,
				child_name,
				registration_number,
				vaccine_name,
				next_due_date
			FROM combined_due
			ORDER BY clinic_id, child_id, vaccine_name, next_due_date, source_rank
		)
		SELECT
			clinic_id,
			clinic_date::text,
			location,
			child_id,
			child_name,
			registration_number,
			vaccine_name,
			next_due_date::text
		FROM dedup_due
		ORDER BY clinic_date ASC, child_id ASC, vaccine_name ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.ParentDueVaccination
	for rows.Next() {
		var item models.ParentDueVaccination
		err := rows.Scan(
			&item.ClinicId,
			&item.ClinicDate,
			&item.ClinicLocation,
			&item.ChildId,
			&item.ChildName,
			&item.RegistrationNumber,
			&item.VaccineName,
			&item.NextDueDate,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, rows.Err()
}
