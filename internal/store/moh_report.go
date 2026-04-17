package store

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MOHReportStore struct {
	pool *pgxpool.Pool
}

func NewMOHReportStore(pool *pgxpool.Pool) *MOHReportStore {
	return &MOHReportStore{pool: pool}
}

// VaccinationCoverageReport groups coverage by GN division with date range
func (s *MOHReportStore) VaccinationCoverageReport(ctx context.Context, startDate, endDate, gnDivision string) ([]map[string]interface{}, error) {
	log.Printf("[VaccinationCoverageReport] Starting report generation - startDate: %s, endDate: %s, gnDivision: %s", startDate, endDate, gnDivision)

	query := `
		SELECT 
			c.gn_division,
			COUNT(DISTINCT c.id) AS total,
			COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END) AS vaccinated,
			CASE 
				WHEN COUNT(DISTINCT c.id) = 0 THEN 0::double precision
				ELSE ROUND(
					(COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END)::numeric / 
					COUNT(DISTINCT c.id)::numeric) * 100, 1
				)::double precision
			END AS coverage
		FROM children c
		LEFT JOIN vaccination_records vr ON vr.child_id = c.id
	`
	args := []interface{}{}
	idx := 1

	conditions := `WHERE 1=1`
	if startDate != "" {
		conditions += fmt.Sprintf(` AND (vr.administered_date >= $%d OR vr.administered_date IS NULL)`, idx)
		args = append(args, startDate)
		idx++
		log.Printf("[VaccinationCoverageReport] Added startDate filter: %s", startDate)
	}
	if endDate != "" {
		conditions += fmt.Sprintf(` AND (vr.administered_date <= $%d OR vr.administered_date IS NULL)`, idx)
		args = append(args, endDate)
		idx++
		log.Printf("[VaccinationCoverageReport] Added endDate filter: %s", endDate)
	}
	if gnDivision != "" {
		conditions += fmt.Sprintf(` AND c.gn_division = $%d`, idx)
		args = append(args, gnDivision)
		idx++
		log.Printf("[VaccinationCoverageReport] Added gnDivision filter: %s", gnDivision)
	}

	query += conditions + ` GROUP BY c.gn_division ORDER BY c.gn_division ASC`

	log.Printf("[VaccinationCoverageReport] Executing query with args: %v", args)
	log.Printf("[VaccinationCoverageReport] Query: %s", query)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[VaccinationCoverageReport] Database error: %v", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var gnDiv string
		var total, vaccinated int
		var coverage float64
		if err := rows.Scan(&gnDiv, &total, &vaccinated, &coverage); err != nil {
			log.Printf("[VaccinationCoverageReport] Scan error: %v", err)
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, map[string]interface{}{
			"gnDivision": gnDiv,
			"total":      total,
			"vaccinated": vaccinated,
			"coverage":   coverage,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[VaccinationCoverageReport] Rows error: %v", err)
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	log.Printf("[VaccinationCoverageReport] Successfully generated report with %d rows", len(results))
	return results, nil
}

// MissedVaccinationReport lists missed vaccinations with child details
func (s *MOHReportStore) MissedVaccinationReport(ctx context.Context, startDate, endDate, gnDivision string) ([]map[string]interface{}, error) {
	log.Printf("[MissedVaccinationReport] Starting report generation - startDate: %s, endDate: %s, gnDivision: %s", startDate, endDate, gnDivision)

	query := `
		SELECT 
			c.first_name,
			c.last_name,
			c.gn_division,
			v.name,
			vr.next_due_date::text
		FROM vaccination_records vr
		JOIN children c ON c.id = vr.child_id
		JOIN vaccines v ON v.id = vr.vaccine_id
		WHERE vr.next_due_date < CURRENT_DATE
		AND vr.status != 'Completed'
	`
	args := []interface{}{}
	idx := 1

	if startDate != "" {
		query += fmt.Sprintf(` AND vr.next_due_date >= $%d`, idx)
		args = append(args, startDate)
		idx++
		log.Printf("[MissedVaccinationReport] Added startDate filter: %s", startDate)
	}
	if endDate != "" {
		query += fmt.Sprintf(` AND vr.next_due_date <= $%d`, idx)
		args = append(args, endDate)
		idx++
		log.Printf("[MissedVaccinationReport] Added endDate filter: %s", endDate)
	}
	if gnDivision != "" {
		query += fmt.Sprintf(` AND c.gn_division = $%d`, idx)
		args = append(args, gnDivision)
		idx++
		log.Printf("[MissedVaccinationReport] Added gnDivision filter: %s", gnDivision)
	}

	query += ` ORDER BY vr.next_due_date ASC`

	log.Printf("[MissedVaccinationReport] Executing query with args: %v", args)
	log.Printf("[MissedVaccinationReport] Query: %s", query)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[MissedVaccinationReport] Database error: %v", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var firstName, lastName, gnDiv, vaccineName, dueDate string
		if err := rows.Scan(&firstName, &lastName, &gnDiv, &vaccineName, &dueDate); err != nil {
			log.Printf("[MissedVaccinationReport] Scan error: %v", err)
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, map[string]interface{}{
			"childName":  firstName + " " + lastName,
			"gnDivision": gnDiv,
			"vaccine":    vaccineName,
			"dueDate":    dueDate,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[MissedVaccinationReport] Rows error: %v", err)
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	log.Printf("[MissedVaccinationReport] Successfully generated report with %d rows", len(results))
	return results, nil
}

// PHMPerformanceReport returns detailed performance metrics for each PHM
func (s *MOHReportStore) PHMPerformanceReport(ctx context.Context, startDate, endDate, gnDivision string) ([]map[string]interface{}, error) {
	log.Printf("[PHMPerformanceReport] Starting report generation - startDate: %s, endDate: %s, gnDivision: %s", startDate, endDate, gnDivision)

	query := `
		SELECT 
			u.name AS phm,
			u.assigned_area AS area,
			COUNT(DISTINCT c.id) AS total,
			COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END) AS vaccinated,
			COUNT(DISTINCT CASE WHEN vr.next_due_date < CURRENT_DATE AND vr.status != 'Completed' THEN c.id END) AS missed,
			CASE 
				WHEN COUNT(DISTINCT c.id) = 0 THEN 0::double precision
				ELSE ROUND(
					(COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END)::numeric / 
					COUNT(DISTINCT c.id)::numeric) * 100, 1
				)::double precision
			END AS coverage
		FROM users u
		LEFT JOIN children c ON c.gn_division = u.assigned_area
		LEFT JOIN vaccination_records vr ON vr.child_id = c.id
		WHERE u.role = 'phm' AND u.assigned_area IS NOT NULL
	`
	args := []interface{}{}
	idx := 1

	if startDate != "" {
		query += fmt.Sprintf(` AND (vr.administered_date >= $%d OR vr.administered_date IS NULL)`, idx)
		args = append(args, startDate)
		idx++
		log.Printf("[PHMPerformanceReport] Added startDate filter: %s", startDate)
	}
	if endDate != "" {
		query += fmt.Sprintf(` AND (vr.administered_date <= $%d OR vr.administered_date IS NULL)`, idx)
		args = append(args, endDate)
		idx++
		log.Printf("[PHMPerformanceReport] Added endDate filter: %s", endDate)
	}
	if gnDivision != "" {
		query += fmt.Sprintf(` AND u.assigned_area = $%d`, idx)
		args = append(args, gnDivision)
		idx++
		log.Printf("[PHMPerformanceReport] Added gnDivision filter: %s", gnDivision)
	}

	query += ` GROUP BY u.id, u.name, u.assigned_area ORDER BY u.name ASC`

	log.Printf("[PHMPerformanceReport] Executing query with args: %v", args)
	log.Printf("[PHMPerformanceReport] Query: %s", query)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[PHMPerformanceReport] Database error: %v", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var phm, area string
		var total, vaccinated, missed int
		var coverage float64
		if err := rows.Scan(&phm, &area, &total, &vaccinated, &missed, &coverage); err != nil {
			log.Printf("[PHMPerformanceReport] Scan error: %v", err)
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, map[string]interface{}{
			"phm":        phm,
			"area":       area,
			"total":      total,
			"vaccinated": vaccinated,
			"missed":     missed,
			"coverage":   coverage,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[PHMPerformanceReport] Rows error: %v", err)
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	log.Printf("[PHMPerformanceReport] Successfully generated report with %d rows", len(results))
	return results, nil
}

// AuditReport returns audit logs with date range and role filters
func (s *MOHReportStore) AuditReport(ctx context.Context, startDate, endDate, role, action string) ([]map[string]interface{}, error) {
	log.Printf("[AuditReport] Starting report generation - startDate: %s, endDate: %s, role: %s, action: %s", startDate, endDate, role, action)

	query := `
		SELECT 
			timestamp::text,
			COALESCE(user_name, 'System') as user_name,
			user_role,
			action,
			details
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	idx := 1

	if startDate != "" {
		query += fmt.Sprintf(` AND timestamp >= $%d`, idx)
		args = append(args, startDate)
		idx++
		log.Printf("[AuditReport] Added startDate filter: %s", startDate)
	}
	if endDate != "" {
		query += fmt.Sprintf(` AND timestamp <= $%d`, idx)
		args = append(args, endDate)
		idx++
		log.Printf("[AuditReport] Added endDate filter: %s", endDate)
	}
	if role != "" {
		query += fmt.Sprintf(` AND user_role = $%d`, idx)
		args = append(args, role)
		idx++
		log.Printf("[AuditReport] Added role filter: %s", role)
	}
	if action != "" {
		query += fmt.Sprintf(` AND action = $%d`, idx)
		args = append(args, action)
		idx++
		log.Printf("[AuditReport] Added action filter: %s", action)
	}

	query += ` ORDER BY timestamp DESC`

	log.Printf("[AuditReport] Executing query with args: %v", args)
	log.Printf("[AuditReport] Query: %s", query)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[AuditReport] Database error: %v", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timestamp, userName, userRole, action, details string
		if err := rows.Scan(&timestamp, &userName, &userRole, &action, &details); err != nil {
			log.Printf("[AuditReport] Scan error: %v", err)
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, map[string]interface{}{
			"date":    timestamp,
			"user":    userName,
			"role":    userRole,
			"action":  action,
			"details": details,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[AuditReport] Rows error: %v", err)
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	log.Printf("[AuditReport] Successfully generated report with %d rows", len(results))
	return results, nil
}
