package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VaccinationRecordStore struct {
	pool *pgxpool.Pool
}

func NewVaccinationRecordStore(pool *pgxpool.Pool) *VaccinationRecordStore {
	return &VaccinationRecordStore{pool: pool}
}

func (s *VaccinationRecordStore) Create(ctx context.Context, id, childID, vaccineID, administeredDate, batchNumber,
	administeredBy, location, site string, doseNumber *int, nextDueDate *string, status, notes string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vaccination_records (id, child_id, vaccine_id, administered_date, batch_number, administered_by,
			location, site, dose_number, next_due_date, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, id, childID, vaccineID, administeredDate, batchNumber, administeredBy, location, site, doseNumber, nextDueDate, status, notes)
	return err
}

func (s *VaccinationRecordStore) GetByID(ctx context.Context, recordID string) (*models.VaccinationRecord, error) {
	var r models.VaccinationRecord
	var dose *int
	var nextDue *string
	err := s.pool.QueryRow(ctx, `
		SELECT vr.id, vr.child_id, v.name, vr.administered_date::text, vr.batch_number, vr.administered_by,
		       vr.location, vr.site, vr.dose_number, vr.next_due_date::text, vr.status, vr.notes, vr.created_at
		FROM vaccination_records vr JOIN vaccines v ON v.id = vr.vaccine_id WHERE vr.id = $1
	`, recordID).Scan(&r.RecordId, &r.ChildId, &r.VaccineName, &r.AdministeredDate, &r.BatchNumber, &r.AdministeredBy,
		&r.Location, &r.Site, &dose, &nextDue, &r.Status, &r.Notes, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.VaccineId = ""
	r.DoseNumber = dose
	r.NextDueDate = nextDue
	return &r, nil
}

func (s *VaccinationRecordStore) ByChildID(ctx context.Context, childID string) ([]models.VaccinationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vr.id, vr.child_id, vr.vaccine_id, v.name, vr.administered_date::text, vr.batch_number, vr.administered_by,
		       vr.location, vr.site, vr.dose_number, vr.next_due_date::text, vr.status, vr.notes, vr.created_at
		FROM vaccination_records vr JOIN vaccines v ON v.id = vr.vaccine_id WHERE vr.child_id = $1 ORDER BY vr.administered_date DESC
	`, childID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanVaccinationRecords(rows)
}

func (s *VaccinationRecordStore) ListMOH(ctx context.Context, areaCode, vaccineID, status, startDate, endDate string, page, limit int) (total int, list []models.VaccinationRecord, err error) {
	base := `FROM vaccination_records vr JOIN vaccines v ON v.id = vr.vaccine_id JOIN children c ON c.id = vr.child_id WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if areaCode != "" {
		base += fmt.Sprintf(` AND c.area_code = $%d`, idx)
		args = append(args, areaCode)
		idx++
	}
	if vaccineID != "" {
		base += fmt.Sprintf(` AND vr.vaccine_id = $%d`, idx)
		args = append(args, vaccineID)
		idx++
	}
	if status != "" {
		base += fmt.Sprintf(` AND vr.status = $%d`, idx)
		args = append(args, status)
		idx++
	}
	if startDate != "" {
		base += fmt.Sprintf(` AND vr.administered_date >= $%d`, idx)
		args = append(args, startDate)
		idx++
	}
	if endDate != "" {
		base += fmt.Sprintf(` AND vr.administered_date <= $%d`, idx)
		args = append(args, endDate)
		idx++
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) `+base, args...).Scan(&total)
	if err != nil {
		return 0, nil, err
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := s.pool.Query(ctx, `
		SELECT vr.id, vr.child_id, vr.vaccine_id, v.name, vr.administered_date::text, vr.batch_number, vr.administered_by,
		       vr.location, vr.site, vr.dose_number, vr.next_due_date::text, vr.status, vr.notes, vr.created_at
		`+base+` ORDER BY vr.administered_date DESC LIMIT $`+fmt.Sprint(idx)+` OFFSET $`+fmt.Sprint(idx+1), args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	list, err = scanVaccinationRecords(rows)
	return total, list, err
}

func (s *VaccinationRecordStore) Update(ctx context.Context, recordID string, vaccineID, administeredDate, batchNumber,
	administeredBy, location, site string, doseNumber *int, nextDueDate *string, status, notes string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE vaccination_records SET vaccine_id = COALESCE(NULLIF($2,''), vaccine_id), administered_date = COALESCE($3::date, administered_date),
			batch_number = COALESCE(NULLIF($4,''), batch_number), administered_by = COALESCE(NULLIF($5,''), administered_by),
			location = COALESCE(NULLIF($6,''), location), site = COALESCE(NULLIF($7,''), site),
			dose_number = COALESCE($8, dose_number), next_due_date = COALESCE($9::date, next_due_date),
			status = COALESCE(NULLIF($10,''), status), notes = COALESCE(NULLIF($11,''), notes)
		WHERE id = $1
	`, recordID, vaccineID, administeredDate, batchNumber, administeredBy, location, site, doseNumber, nextDueDate, status, notes)
	return err
}

func (s *VaccinationRecordStore) Delete(ctx context.Context, recordID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM vaccination_records WHERE id = $1`, recordID)
	return err
}

func scanVaccinationRecords(rows pgx.Rows) ([]models.VaccinationRecord, error) {
	var list []models.VaccinationRecord
	for rows.Next() {
		var r models.VaccinationRecord
		var dose *int
		var nextDue *string
		err := rows.Scan(&r.RecordId, &r.ChildId, &r.VaccineId, &r.VaccineName, &r.AdministeredDate, &r.BatchNumber, &r.AdministeredBy,
			&r.Location, &r.Site, &dose, &nextDue, &r.Status, &r.Notes, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		r.DoseNumber = dose
		r.NextDueDate = nextDue
		list = append(list, r)
	}
	return list, rows.Err()
}
