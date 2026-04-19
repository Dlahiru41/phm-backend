package store

import (
	"context"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleStore struct {
	pool *pgxpool.Pool
}

func NewScheduleStore(pool *pgxpool.Pool) *ScheduleStore { return &ScheduleStore{pool: pool} }

func (s *ScheduleStore) Create(ctx context.Context, id, childID, vaccineID, scheduledDate, dueDate string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vaccination_schedules (id, child_id, vaccine_id, scheduled_date, due_date, status)
		VALUES ($1, $2, $3, $4, $5, 'scheduled')
	`, id, childID, vaccineID, scheduledDate, dueDate)
	return err
}

func (s *ScheduleStore) ByChildID(ctx context.Context, childID string) ([]models.VaccinationSchedule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT vs.id, vs.child_id, vs.vaccine_id, v.name, vs.scheduled_date::text, vs.due_date::text, vs.status, vs.reminder_sent
		FROM vaccination_schedules vs JOIN vaccines v ON v.id = vs.vaccine_id WHERE vs.child_id = $1 ORDER BY vs.due_date
	`, childID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.VaccinationSchedule
	for rows.Next() {
		var sch models.VaccinationSchedule
		err := rows.Scan(&sch.ScheduleId, &sch.ChildId, &sch.VaccineId, &sch.VaccineName, &sch.ScheduledDate, &sch.DueDate, &sch.Status, &sch.ReminderSent)
		if err != nil {
			return nil, err
		}
		list = append(list, sch)
	}
	return list, rows.Err()
}

func (s *ScheduleStore) GetByID(ctx context.Context, scheduleID string) (*models.VaccinationSchedule, error) {
	var sch models.VaccinationSchedule
	err := s.pool.QueryRow(ctx, `
		SELECT vs.id, vs.child_id, vs.vaccine_id, v.name, vs.scheduled_date::text, vs.due_date::text, vs.status, vs.reminder_sent
		FROM vaccination_schedules vs JOIN vaccines v ON v.id = vs.vaccine_id WHERE vs.id = $1
	`, scheduleID).Scan(&sch.ScheduleId, &sch.ChildId, &sch.VaccineId, &sch.VaccineName, &sch.ScheduledDate, &sch.DueDate, &sch.Status, &sch.ReminderSent)
	if err != nil {
		return nil, err
	}
	return &sch, nil
}

func (s *ScheduleStore) UpdateStatus(ctx context.Context, scheduleID, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE vaccination_schedules SET status = $2 WHERE id = $1`, scheduleID, status)
	return err
}

func (s *ScheduleStore) SetReminderSent(ctx context.Context, scheduleID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE vaccination_schedules SET reminder_sent = true WHERE id = $1`, scheduleID)
	return err
}

func (s *ScheduleStore) ListDueForPHM(ctx context.Context, phmID string) ([]models.PHMDueVaccination, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			vs.id,
			c.id,
			TRIM(CONCAT(c.first_name, ' ', c.last_name)) AS child_name,
			c.registration_number,
			vs.vaccine_id,
			v.name,
			vs.due_date::text,
			vs.status,
			c.parent_id,
			NULLIF(TRIM(COALESCE(u.phone_number, c.parent_whatsapp_number, '')), '') AS parent_phone,
			vs.reminder_sent,
			vs.missed_notified
		FROM vaccination_schedules vs
		JOIN children c ON c.id = vs.child_id
		JOIN vaccines v ON v.id = vs.vaccine_id
		LEFT JOIN users u ON u.id = c.parent_id
		WHERE c.registered_by = $1
		  AND vs.due_date <= CURRENT_DATE
		  AND vs.status IN ('pending', 'scheduled', 'missed')
		ORDER BY vs.due_date ASC, c.created_at ASC
	`, phmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.PHMDueVaccination
	for rows.Next() {
		var item models.PHMDueVaccination
		if err := rows.Scan(
			&item.ScheduleId,
			&item.ChildId,
			&item.ChildName,
			&item.RegistrationNumber,
			&item.VaccineId,
			&item.VaccineName,
			&item.DueDate,
			&item.Status,
			&item.ParentId,
			&item.ParentPhone,
			&item.ReminderSent,
			&item.MissedNotified,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *ScheduleStore) MarkMissedDueVaccinations(ctx context.Context, phmID string) ([]models.PHMDueVaccination, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE vaccination_schedules vs
		SET status = 'missed', updated_at = NOW()
		FROM children c
		LEFT JOIN users u ON u.id = c.parent_id
		WHERE vs.child_id = c.id
		  AND c.registered_by = $1
		  AND vs.due_date < CURRENT_DATE
		  AND vs.status IN ('pending', 'scheduled')
		RETURNING
			vs.id,
			c.id,
			TRIM(CONCAT(c.first_name, ' ', c.last_name)) AS child_name,
			c.registration_number,
			vs.vaccine_id,
			(SELECT v.name FROM vaccines v WHERE v.id = vs.vaccine_id) AS vaccine_name,
			vs.due_date::text,
			vs.status,
			c.parent_id,
			NULLIF(TRIM(COALESCE(u.phone_number, c.parent_whatsapp_number, '')), '') AS parent_phone,
			vs.reminder_sent,
			vs.missed_notified
	`, phmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.PHMDueVaccination
	for rows.Next() {
		var item models.PHMDueVaccination
		if err := rows.Scan(
			&item.ScheduleId,
			&item.ChildId,
			&item.ChildName,
			&item.RegistrationNumber,
			&item.VaccineId,
			&item.VaccineName,
			&item.DueDate,
			&item.Status,
			&item.ParentId,
			&item.ParentPhone,
			&item.ReminderSent,
			&item.MissedNotified,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *ScheduleStore) SetMissedNotified(ctx context.Context, scheduleID string) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE vaccination_schedules SET missed_notified = true, updated_at = NOW() WHERE id = $1`, scheduleID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *ScheduleStore) GetNotificationContextByScheduleID(ctx context.Context, scheduleID string) (*models.PHMDueVaccination, error) {
	var item models.PHMDueVaccination
	err := s.pool.QueryRow(ctx, `
		SELECT
			vs.id,
			c.id,
			TRIM(CONCAT(c.first_name, ' ', c.last_name)) AS child_name,
			c.registration_number,
			vs.vaccine_id,
			v.name,
			vs.due_date::text,
			vs.status,
			c.parent_id,
			NULLIF(TRIM(COALESCE(u.phone_number, c.parent_whatsapp_number, '')), '') AS parent_phone,
			vs.reminder_sent,
			vs.missed_notified
		FROM vaccination_schedules vs
		JOIN children c ON c.id = vs.child_id
		JOIN vaccines v ON v.id = vs.vaccine_id
		LEFT JOIN users u ON u.id = c.parent_id
		WHERE vs.id = $1
	`, scheduleID).Scan(
		&item.ScheduleId,
		&item.ChildId,
		&item.ChildName,
		&item.RegistrationNumber,
		&item.VaccineId,
		&item.VaccineName,
		&item.DueDate,
		&item.Status,
		&item.ParentId,
		&item.ParentPhone,
		&item.ReminderSent,
		&item.MissedNotified,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}
