package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MOHReportStore struct {
	pool *pgxpool.Pool
}

type mohSystemSummary struct {
	TotalChildren           int     `json:"totalChildren"`
	LinkedChildren          int     `json:"linkedChildren"`
	TotalPHMUsers           int     `json:"totalPhmUsers"`
	TotalMOHUsers           int     `json:"totalMohUsers"`
	TotalVaccinationRecords int     `json:"totalVaccinationRecords"`
	AdministeredRecords     int     `json:"administeredRecords"`
	PendingRecords          int     `json:"pendingRecords"`
	MissedRecords           int     `json:"missedRecords"`
	CancelledRecords        int     `json:"cancelledRecords"`
	CoveragePct             float64 `json:"coveragePct"`
	UpcomingDueNext7Days    int     `json:"upcomingDueNext7Days"`
	OverdueRecords          int     `json:"overdueRecords"`
	PendingSchedules        int     `json:"pendingSchedules"`
	CompletedSchedules      int     `json:"completedSchedules"`
	ScheduledClinics        int     `json:"scheduledClinics"`
	CompletedClinics        int     `json:"completedClinics"`
	UnreadNotifications     int     `json:"unreadNotifications"`
	AuditEventsLast30Days   int     `json:"auditEventsLast30Days"`
}

type mohGNDivisionMetric struct {
	GNDivision         string  `json:"gnDivision"`
	RegisteredChildren int     `json:"registeredChildren"`
	LinkedChildren     int     `json:"linkedChildren"`
	VaccinatedChildren int     `json:"vaccinatedChildren"`
	MissedChildren     int     `json:"missedChildren"`
	OverdueRecords     int     `json:"overdueRecords"`
	CoveragePct        float64 `json:"coveragePct"`
}

type mohVaccineMetric struct {
	VaccineID         string  `json:"vaccineId"`
	VaccineName       string  `json:"vaccineName"`
	TotalDoses        int     `json:"totalDoses"`
	AdministeredDoses int     `json:"administeredDoses"`
	PendingDoses      int     `json:"pendingDoses"`
	MissedDoses       int     `json:"missedDoses"`
	CancelledDoses    int     `json:"cancelledDoses"`
	CompletionRatePct float64 `json:"completionRatePct"`
}

type mohMonthlyTrend struct {
	Month             string `json:"month"`
	NewChildren       int    `json:"newChildren"`
	AdministeredDoses int    `json:"administeredDoses"`
	MissedDoses       int    `json:"missedDoses"`
	CompletedClinics  int    `json:"completedClinics"`
	NotificationsSent int    `json:"notificationsSent"`
	AuditEvents       int    `json:"auditEvents"`
}

type mohDataQuality struct {
	ChildrenWithoutGNDivision     int `json:"childrenWithoutGnDivision"`
	ChildrenWithoutLinkedParent   int `json:"childrenWithoutLinkedParent"`
	ChildrenWithoutWhatsAppNumber int `json:"childrenWithoutWhatsAppNumber"`
	PendingRecordsWithoutDueDate  int `json:"pendingRecordsWithoutDueDate"`
	OverduePendingSchedules       int `json:"overduePendingSchedules"`
}

type mohTableFootprint struct {
	Users                int `json:"users"`
	Children             int `json:"children"`
	Vaccines             int `json:"vaccines"`
	VaccinationRecords   int `json:"vaccinationRecords"`
	VaccinationSchedules int `json:"vaccinationSchedules"`
	ClinicSchedules      int `json:"clinicSchedules"`
	ClinicChildren       int `json:"clinicChildren"`
	Notifications        int `json:"notifications"`
	AuditLogs            int `json:"auditLogs"`
	Reports              int `json:"reports"`
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

// SystemOverviewReport returns a system-level, MOH-focused deep report with summary KPIs and structural diagnostics.
func (s *MOHReportStore) SystemOverviewReport(ctx context.Context, startDate, endDate, gnDivision string, trendMonths int) (map[string]interface{}, error) {
	if trendMonths <= 0 {
		trendMonths = 12
	}

	childFilter := "1=1"
	childArgs := []interface{}{}
	childIdx := 1
	if gnDivision != "" {
		childFilter += fmt.Sprintf(" AND c.gn_division = $%d", childIdx)
		childArgs = append(childArgs, gnDivision)
		childIdx++
	}

	recordFilter := "1=1"
	recordArgs := append([]interface{}{}, childArgs...)
	recordIdx := childIdx
	if startDate != "" {
		recordFilter += fmt.Sprintf(" AND vr.administered_date >= $%d", recordIdx)
		recordArgs = append(recordArgs, startDate)
		recordIdx++
	}
	if endDate != "" {
		recordFilter += fmt.Sprintf(" AND vr.administered_date <= $%d", recordIdx)
		recordArgs = append(recordArgs, endDate)
		recordIdx++
	}

	scheduleFilter := "1=1"
	scheduleIdx := childIdx
	if startDate != "" {
		scheduleFilter += fmt.Sprintf(" AND vs.due_date >= $%d", scheduleIdx)
		scheduleIdx++
	}
	if endDate != "" {
		scheduleFilter += fmt.Sprintf(" AND vs.due_date <= $%d", scheduleIdx)
		scheduleIdx++
	}

	clinicFilter := "1=1"
	clinicIdx := 1
	if gnDivision != "" {
		clinicFilter += fmt.Sprintf(" AND cs.gn_division = $%d", clinicIdx)
		clinicIdx++
	}
	if startDate != "" {
		clinicFilter += fmt.Sprintf(" AND cs.clinic_date >= $%d", clinicIdx)
		clinicIdx++
	}
	if endDate != "" {
		clinicFilter += fmt.Sprintf(" AND cs.clinic_date <= $%d", clinicIdx)
		clinicIdx++
	}

	phmScope := ""
	if gnDivision != "" {
		phmScope = " AND u.assigned_area = $1"
	}

	summaryQuery := `
		WITH filtered_children AS (
			SELECT c.id, c.parent_id
			FROM children c
			WHERE ` + childFilter + `
		),
		filtered_records AS (
			SELECT vr.id, vr.child_id, vr.status, vr.next_due_date
			FROM vaccination_records vr
			JOIN filtered_children fc ON fc.id = vr.child_id
			WHERE ` + recordFilter + `
		),
		filtered_schedules AS (
			SELECT vs.id, vs.status, vs.due_date
			FROM vaccination_schedules vs
			JOIN filtered_children fc ON fc.id = vs.child_id
			WHERE ` + scheduleFilter + `
		),
		filtered_clinics AS (
			SELECT cs.id, cs.status
			FROM clinic_schedules cs
			WHERE ` + clinicFilter + `
		)
		SELECT
			(SELECT COUNT(*) FROM filtered_children) AS total_children,
			(SELECT COUNT(*) FROM filtered_children WHERE parent_id IS NOT NULL) AS linked_children,
			(SELECT COUNT(*) FROM users u WHERE u.role = 'phm'` + phmScope + `) AS total_phm_users,
			(SELECT COUNT(*) FROM users u WHERE u.role = 'moh') AS total_moh_users,
			(SELECT COUNT(*) FROM filtered_records) AS total_vaccination_records,
			(SELECT COUNT(*) FROM filtered_records WHERE status = 'administered') AS administered_records,
			(SELECT COUNT(*) FROM filtered_records WHERE status = 'pending') AS pending_records,
			(SELECT COUNT(*) FROM filtered_records WHERE status = 'missed') AS missed_records,
			(SELECT COUNT(*) FROM filtered_records WHERE status = 'cancelled') AS cancelled_records,
			CASE
				WHEN (SELECT COUNT(*) FROM filtered_children) = 0 THEN 0::double precision
				ELSE ROUND((
					(SELECT COUNT(DISTINCT child_id) FROM filtered_records WHERE status = 'administered')::numeric /
					(SELECT COUNT(*) FROM filtered_children)::numeric
				) * 100, 1)::double precision
			END AS coverage_pct,
			(SELECT COUNT(*) FROM filtered_records WHERE status IN ('pending', 'missed') AND next_due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 day') AS upcoming_due_next_7_days,
			(SELECT COUNT(*) FROM filtered_records WHERE status IN ('pending', 'missed') AND next_due_date < CURRENT_DATE) AS overdue_records,
			(SELECT COUNT(*) FROM filtered_schedules WHERE status = 'pending') AS pending_schedules,
			(SELECT COUNT(*) FROM filtered_schedules WHERE status = 'completed') AS completed_schedules,
			(SELECT COUNT(*) FROM filtered_clinics WHERE status = 'scheduled') AS scheduled_clinics,
			(SELECT COUNT(*) FROM filtered_clinics WHERE status = 'completed') AS completed_clinics,
			(SELECT COUNT(*) FROM notifications WHERE is_read = false) AS unread_notifications,
			(SELECT COUNT(*) FROM audit_logs WHERE timestamp >= NOW() - INTERVAL '30 day') AS audit_events_last_30_days
	`

	combinedArgs := recordArgs

	var summary mohSystemSummary
	if err := s.pool.QueryRow(ctx, summaryQuery, combinedArgs...).Scan(
		&summary.TotalChildren,
		&summary.LinkedChildren,
		&summary.TotalPHMUsers,
		&summary.TotalMOHUsers,
		&summary.TotalVaccinationRecords,
		&summary.AdministeredRecords,
		&summary.PendingRecords,
		&summary.MissedRecords,
		&summary.CancelledRecords,
		&summary.CoveragePct,
		&summary.UpcomingDueNext7Days,
		&summary.OverdueRecords,
		&summary.PendingSchedules,
		&summary.CompletedSchedules,
		&summary.ScheduledClinics,
		&summary.CompletedClinics,
		&summary.UnreadNotifications,
		&summary.AuditEventsLast30Days,
	); err != nil {
		return nil, fmt.Errorf("system summary query failed: %w", err)
	}

	gnQuery := `
		WITH filtered_children AS (
			SELECT c.id, c.parent_id, COALESCE(NULLIF(c.gn_division, ''), 'UNASSIGNED') AS gn_division
			FROM children c
			WHERE ` + childFilter + `
		),
		filtered_records AS (
			SELECT vr.child_id, vr.status, vr.next_due_date
			FROM vaccination_records vr
			JOIN filtered_children fc ON fc.id = vr.child_id
			WHERE ` + recordFilter + `
		)
		SELECT
			fc.gn_division,
			COUNT(DISTINCT fc.id) AS registered_children,
			COUNT(DISTINCT CASE WHEN fc.parent_id IS NOT NULL THEN fc.id END) AS linked_children,
			COUNT(DISTINCT CASE WHEN fr.status = 'administered' THEN fc.id END) AS vaccinated_children,
			COUNT(DISTINCT CASE WHEN fr.status = 'missed' THEN fc.id END) AS missed_children,
			COUNT(CASE WHEN fr.status IN ('pending', 'missed') AND fr.next_due_date < CURRENT_DATE THEN 1 END) AS overdue_records,
			CASE
				WHEN COUNT(DISTINCT fc.id) = 0 THEN 0::double precision
				ELSE ROUND((COUNT(DISTINCT CASE WHEN fr.status = 'administered' THEN fc.id END)::numeric / COUNT(DISTINCT fc.id)::numeric) * 100, 1)::double precision
			END AS coverage_pct
		FROM filtered_children fc
		LEFT JOIN filtered_records fr ON fr.child_id = fc.id
		GROUP BY fc.gn_division
		ORDER BY coverage_pct DESC, registered_children DESC
	`

	gnRows, err := s.pool.Query(ctx, gnQuery, recordArgs...)
	if err != nil {
		return nil, fmt.Errorf("gn division breakdown query failed: %w", err)
	}
	defer gnRows.Close()

	gnStats := make([]mohGNDivisionMetric, 0)
	for gnRows.Next() {
		var row mohGNDivisionMetric
		if err := gnRows.Scan(
			&row.GNDivision,
			&row.RegisteredChildren,
			&row.LinkedChildren,
			&row.VaccinatedChildren,
			&row.MissedChildren,
			&row.OverdueRecords,
			&row.CoveragePct,
		); err != nil {
			return nil, fmt.Errorf("gn division breakdown scan failed: %w", err)
		}
		gnStats = append(gnStats, row)
	}
	if err := gnRows.Err(); err != nil {
		return nil, fmt.Errorf("gn division breakdown rows failed: %w", err)
	}

	vaccineQuery := `
		WITH filtered_children AS (
			SELECT c.id
			FROM children c
			WHERE ` + childFilter + `
		),
		filtered_records AS (
			SELECT vr.vaccine_id, vr.status
			FROM vaccination_records vr
			JOIN filtered_children fc ON fc.id = vr.child_id
			WHERE ` + recordFilter + `
		)
		SELECT
			fr.vaccine_id,
			COALESCE(v.name, 'Unknown') AS vaccine_name,
			COUNT(*) AS total_doses,
			COUNT(*) FILTER (WHERE fr.status = 'administered') AS administered_doses,
			COUNT(*) FILTER (WHERE fr.status = 'pending') AS pending_doses,
			COUNT(*) FILTER (WHERE fr.status = 'missed') AS missed_doses,
			COUNT(*) FILTER (WHERE fr.status = 'cancelled') AS cancelled_doses,
			CASE WHEN COUNT(*) = 0 THEN 0::double precision
				ELSE ROUND((COUNT(*) FILTER (WHERE fr.status = 'administered')::numeric / COUNT(*)::numeric) * 100, 1)::double precision
			END AS completion_rate_pct
		FROM filtered_records fr
		LEFT JOIN vaccines v ON v.id = fr.vaccine_id
		GROUP BY fr.vaccine_id, v.name
		ORDER BY total_doses DESC, vaccine_name ASC
		LIMIT 20
	`

	vaccineRows, err := s.pool.Query(ctx, vaccineQuery, recordArgs...)
	if err != nil {
		return nil, fmt.Errorf("vaccine performance query failed: %w", err)
	}
	defer vaccineRows.Close()

	vaccineStats := make([]mohVaccineMetric, 0)
	for vaccineRows.Next() {
		var row mohVaccineMetric
		if err := vaccineRows.Scan(
			&row.VaccineID,
			&row.VaccineName,
			&row.TotalDoses,
			&row.AdministeredDoses,
			&row.PendingDoses,
			&row.MissedDoses,
			&row.CancelledDoses,
			&row.CompletionRatePct,
		); err != nil {
			return nil, fmt.Errorf("vaccine performance scan failed: %w", err)
		}
		vaccineStats = append(vaccineStats, row)
	}
	if err := vaccineRows.Err(); err != nil {
		return nil, fmt.Errorf("vaccine performance rows failed: %w", err)
	}

	trendQuery := `
		WITH months AS (
			SELECT generate_series(
				date_trunc('month', CURRENT_DATE) - (($1::int - 1) * INTERVAL '1 month'),
				date_trunc('month', CURRENT_DATE),
				INTERVAL '1 month'
			) AS month_start
		),
		children_monthly AS (
			SELECT date_trunc('month', c.created_at) AS month_start, COUNT(*) AS new_children
			FROM children c
			WHERE 1=1 ` + trendChildClause(gnDivision, 2) + `
			GROUP BY 1
		),
		records_monthly AS (
			SELECT
				date_trunc('month', vr.administered_date::timestamp) AS month_start,
				COUNT(*) FILTER (WHERE vr.status = 'administered') AS administered_doses,
				COUNT(*) FILTER (WHERE vr.status = 'missed') AS missed_doses
			FROM vaccination_records vr
			JOIN children c ON c.id = vr.child_id
			WHERE vr.administered_date >= (SELECT MIN(month_start) FROM months)
			AND vr.administered_date < (SELECT MAX(month_start) + INTERVAL '1 month' FROM months)
			` + trendRecordClause(gnDivision, 2) + `
			GROUP BY 1
		),
		clinics_monthly AS (
			SELECT date_trunc('month', cs.clinic_date::timestamp) AS month_start, COUNT(*) FILTER (WHERE cs.status = 'completed') AS completed_clinics
			FROM clinic_schedules cs
			WHERE cs.clinic_date >= (SELECT MIN(month_start) FROM months)
			AND cs.clinic_date < (SELECT MAX(month_start) + INTERVAL '1 month' FROM months)
			` + trendClinicClause(gnDivision, 2) + `
			GROUP BY 1
		),
		notifications_monthly AS (
			SELECT date_trunc('month', n.sent_date) AS month_start, COUNT(*) AS notifications_sent
			FROM notifications n
			WHERE n.sent_date >= (SELECT MIN(month_start) FROM months)
			AND n.sent_date < (SELECT MAX(month_start) + INTERVAL '1 month' FROM months)
			GROUP BY 1
		),
		audit_monthly AS (
			SELECT date_trunc('month', a.timestamp) AS month_start, COUNT(*) AS audit_events
			FROM audit_logs a
			WHERE a.timestamp >= (SELECT MIN(month_start) FROM months)
			AND a.timestamp < (SELECT MAX(month_start) + INTERVAL '1 month' FROM months)
			GROUP BY 1
		)
		SELECT
			to_char(m.month_start, 'YYYY-MM') AS month,
			COALESCE(cm.new_children, 0) AS new_children,
			COALESCE(rm.administered_doses, 0) AS administered_doses,
			COALESCE(rm.missed_doses, 0) AS missed_doses,
			COALESCE(clm.completed_clinics, 0) AS completed_clinics,
			COALESCE(nm.notifications_sent, 0) AS notifications_sent,
			COALESCE(am.audit_events, 0) AS audit_events
		FROM months m
		LEFT JOIN children_monthly cm ON cm.month_start = m.month_start
		LEFT JOIN records_monthly rm ON rm.month_start = m.month_start
		LEFT JOIN clinics_monthly clm ON clm.month_start = m.month_start
		LEFT JOIN notifications_monthly nm ON nm.month_start = m.month_start
		LEFT JOIN audit_monthly am ON am.month_start = m.month_start
		ORDER BY m.month_start ASC
	`

	trendArgs := []interface{}{trendMonths}
	if gnDivision != "" {
		trendArgs = append(trendArgs, gnDivision)
	}
	trendRows, err := s.pool.Query(ctx, trendQuery, trendArgs...)
	if err != nil {
		return nil, fmt.Errorf("monthly trend query failed: %w", err)
	}
	defer trendRows.Close()

	trend := make([]mohMonthlyTrend, 0, trendMonths)
	for trendRows.Next() {
		var row mohMonthlyTrend
		if err := trendRows.Scan(
			&row.Month,
			&row.NewChildren,
			&row.AdministeredDoses,
			&row.MissedDoses,
			&row.CompletedClinics,
			&row.NotificationsSent,
			&row.AuditEvents,
		); err != nil {
			return nil, fmt.Errorf("monthly trend scan failed: %w", err)
		}
		trend = append(trend, row)
	}
	if err := trendRows.Err(); err != nil {
		return nil, fmt.Errorf("monthly trend rows failed: %w", err)
	}

	qualityQuery := `
		WITH filtered_children AS (
			SELECT c.id, c.parent_id, c.gn_division, c.parent_whatsapp_number
			FROM children c
			WHERE ` + childFilter + `
		),
		all_scoped_records AS (
			SELECT vr.status, vr.next_due_date
			FROM vaccination_records vr
			JOIN filtered_children fc ON fc.id = vr.child_id
		),
		all_scoped_schedules AS (
			SELECT vs.status, vs.due_date
			FROM vaccination_schedules vs
			JOIN filtered_children fc ON fc.id = vs.child_id
		)
		SELECT
			COUNT(*) FILTER (WHERE COALESCE(TRIM(gn_division), '') = '') AS children_without_gn_division,
			COUNT(*) FILTER (WHERE parent_id IS NULL) AS children_without_linked_parent,
			COUNT(*) FILTER (WHERE COALESCE(TRIM(parent_whatsapp_number), '') = '') AS children_without_whatsapp_number,
			(SELECT COUNT(*) FROM all_scoped_records WHERE status = 'pending' AND next_due_date IS NULL) AS pending_records_without_due_date,
			(SELECT COUNT(*) FROM all_scoped_schedules WHERE status = 'pending' AND due_date < CURRENT_DATE) AS overdue_pending_schedules
		FROM filtered_children
	`

	var quality mohDataQuality
	if err := s.pool.QueryRow(ctx, qualityQuery, childArgs...).Scan(
		&quality.ChildrenWithoutGNDivision,
		&quality.ChildrenWithoutLinkedParent,
		&quality.ChildrenWithoutWhatsAppNumber,
		&quality.PendingRecordsWithoutDueDate,
		&quality.OverduePendingSchedules,
	); err != nil {
		return nil, fmt.Errorf("data quality query failed: %w", err)
	}

	footprintQuery := `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM children),
			(SELECT COUNT(*) FROM vaccines),
			(SELECT COUNT(*) FROM vaccination_records),
			(SELECT COUNT(*) FROM vaccination_schedules),
			(SELECT COUNT(*) FROM clinic_schedules),
			(SELECT COUNT(*) FROM clinic_children),
			(SELECT COUNT(*) FROM notifications),
			(SELECT COUNT(*) FROM audit_logs),
			(SELECT COUNT(*) FROM reports)
	`

	var footprint mohTableFootprint
	if err := s.pool.QueryRow(ctx, footprintQuery).Scan(
		&footprint.Users,
		&footprint.Children,
		&footprint.Vaccines,
		&footprint.VaccinationRecords,
		&footprint.VaccinationSchedules,
		&footprint.ClinicSchedules,
		&footprint.ClinicChildren,
		&footprint.Notifications,
		&footprint.AuditLogs,
		&footprint.Reports,
	); err != nil {
		return nil, fmt.Errorf("database footprint query failed: %w", err)
	}

	insights := buildSystemInsights(summary, gnStats, vaccineStats, quality)

	return map[string]interface{}{
		"reportType":  "moh_system_overview",
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
		"filters":     map[string]interface{}{"startDate": startDate, "endDate": endDate, "gnDivision": gnDivision, "trendMonths": trendMonths},
		"summary":     summary,
		"insights":    insights,
		"deepDive": map[string]interface{}{
			"byGNDivision":      gnStats,
			"byVaccine":         vaccineStats,
			"monthlyTrend":      trend,
			"dataQuality":       quality,
			"databaseFootprint": footprint,
		},
	}, nil
}

func trendChildClause(gnDivision string, idx int) string {
	if gnDivision == "" {
		return ""
	}
	return fmt.Sprintf(" AND c.gn_division = $%d", idx)
}

func trendRecordClause(gnDivision string, idx int) string {
	if gnDivision == "" {
		return ""
	}
	return fmt.Sprintf(" AND c.gn_division = $%d", idx)
}

func trendClinicClause(gnDivision string, idx int) string {
	if gnDivision == "" {
		return ""
	}
	return fmt.Sprintf(" AND cs.gn_division = $%d", idx)
}

func buildSystemInsights(summary mohSystemSummary, gnStats []mohGNDivisionMetric, vaccineStats []mohVaccineMetric, quality mohDataQuality) []string {
	insights := []string{}

	insights = append(insights,
		fmt.Sprintf("Coverage is %.1f%% with %d administered records out of %d total records in the selected scope.", summary.CoveragePct, summary.AdministeredRecords, summary.TotalVaccinationRecords),
		fmt.Sprintf("%d records are overdue and %d are due in the next 7 days, highlighting immediate follow-up workload.", summary.OverdueRecords, summary.UpcomingDueNext7Days),
		fmt.Sprintf("Parent linkage is %d/%d children (%.1f%%).", summary.LinkedChildren, summary.TotalChildren, pct(summary.LinkedChildren, summary.TotalChildren)),
	)

	if len(gnStats) > 0 {
		top := gnStats[0]
		bottom := gnStats[len(gnStats)-1]
		insights = append(insights,
			fmt.Sprintf("Top GN division by coverage is %s at %.1f%% (%d children).", top.GNDivision, top.CoveragePct, top.RegisteredChildren),
			fmt.Sprintf("Lowest GN division coverage is %s at %.1f%% with %d overdue records.", bottom.GNDivision, bottom.CoveragePct, bottom.OverdueRecords),
		)
	}

	if len(vaccineStats) > 0 {
		lead := vaccineStats[0]
		insights = append(insights, fmt.Sprintf("Highest administered volume vaccine is %s with %d doses and %.1f%% completion rate.", lead.VaccineName, lead.TotalDoses, lead.CompletionRatePct))
	}

	insights = append(insights,
		fmt.Sprintf("Data quality flags: %d children without GN division, %d without linked parent, and %d pending records without a next due date.", quality.ChildrenWithoutGNDivision, quality.ChildrenWithoutLinkedParent, quality.PendingRecordsWithoutDueDate),
	)

	return insights
}

func pct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) * 100 / float64(den)
}
