package store

import (
	"context"

	"ncvms/internal/models"

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
