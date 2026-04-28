package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MOHDashboardStore struct {
	pool *pgxpool.Pool
}

func NewMOHDashboardStore(pool *pgxpool.Pool) *MOHDashboardStore {
	return &MOHDashboardStore{pool: pool}
}

// TotalChildren returns the total count of children in the system
func (s *MOHDashboardStore) TotalChildren(ctx context.Context) (int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) AS total_children FROM children`).Scan(&total)
	return total, err
}

// ChildrenDistribution groups children by GN division
func (s *MOHDashboardStore) ChildrenDistribution(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT gn_division, COUNT(*) AS total
		FROM children
		GROUP BY gn_division
		ORDER BY total DESC, gn_division ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var gnDivision string
		var total int
		if err := rows.Scan(&gnDivision, &total); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"gnDivision": gnDivision,
			"total":      total,
		})
	}
	return results, rows.Err()
}

// VaccinationCoverage calculates vaccination coverage percentage
func (s *MOHDashboardStore) VaccinationCoverage(ctx context.Context) (int, int, float64, error) {
	var totalChildren int
	var vaccinatedChildren int

	// Get total children
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM children`).Scan(&totalChildren)
	if err != nil {
		return 0, 0, 0, err
	}

	// Get count of distinct children with at least one completed vaccination
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT child_id) AS vaccinated_children
		FROM vaccination_records
		WHERE status = 'Completed'
	`).Scan(&vaccinatedChildren)
	if err != nil {
		return 0, 0, 0, err
	}

	var coverage float64
	if totalChildren > 0 {
		coverage = (float64(vaccinatedChildren) / float64(totalChildren)) * 100
	}

	return totalChildren, vaccinatedChildren, coverage, nil
}

// MissedVaccinations counts overdue vaccinations
func (s *MOHDashboardStore) MissedVaccinations(ctx context.Context) (int, error) {
	var missed int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) AS missed
		FROM vaccination_records
		WHERE next_due_date < CURRENT_DATE
		AND status != 'Completed'
	`).Scan(&missed)
	return missed, err
}

// PHMPerformanceSummary returns performance metrics for each PHM
func (s *MOHDashboardStore) PHMPerformanceSummary(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT 
			u.name AS phm_name,
			u.assigned_area AS gn_division,
			COUNT(DISTINCT c.id) AS total_children,
			COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END) AS vaccinated,
			CASE 
				WHEN COUNT(DISTINCT c.id) = 0 THEN 0::double precision
				ELSE ROUND(
					(COUNT(DISTINCT CASE WHEN vr.status = 'Completed' THEN c.id END)::numeric / 
					COUNT(DISTINCT c.id)::numeric) * 100, 1
				)::double precision
			END AS coverage
		FROM users u
		LEFT JOIN children c ON c.gn_division = u.assigned_area
		LEFT JOIN vaccination_records vr ON vr.child_id = c.id AND vr.status = 'Completed'
		WHERE u.role = 'phm' AND u.assigned_area IS NOT NULL
		GROUP BY u.id, u.name, u.assigned_area
		ORDER BY u.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var phmName, gnDivision *string
		var totalChildren, vaccinated int
		var coverage float64
		if err := rows.Scan(&phmName, &gnDivision, &totalChildren, &vaccinated, &coverage); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"phmName":       phmName,
			"gnDivision":    gnDivision,
			"totalChildren": totalChildren,
			"vaccinated":    vaccinated,
			"coverage":      coverage,
		})
	}
	return results, rows.Err()
}

// RecentChildren returns the latest 10 registered children
func (s *MOHDashboardStore) RecentChildren(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT 
			id, registration_number, first_name, last_name, 
			date_of_birth::text, gender, gn_division, 
			created_at, registered_by
		FROM children
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, regNum, firstName, lastName, dob, gender, gnDiv string
		var createdAt string
		var registeredBy *string
		if err := rows.Scan(&id, &regNum, &firstName, &lastName, &dob, &gender, &gnDiv, &createdAt, &registeredBy); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"childId":            id,
			"registrationNumber": regNum,
			"firstName":          firstName,
			"lastName":           lastName,
			"dateOfBirth":        dob,
			"gender":             gender,
			"gnDivision":         gnDiv,
			"createdAt":          createdAt,
			"registeredBy":       registeredBy,
		})
	}
	return results, rows.Err()
}

// AreaSummary returns comprehensive statistics for a specific area assigned to a PHM
func (s *MOHDashboardStore) AreaSummary(ctx context.Context, assignedArea string) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// Total children in area
	var totalChildren int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM children WHERE gn_division = $1
	`, assignedArea).Scan(&totalChildren)
	if err != nil {
		return nil, err
	}
	summary["totalChildren"] = totalChildren

	// Vaccinated children count (have at least one vaccination record with status 'administered')
	var vaccinatedCount int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT child_id) FROM vaccination_records 
		WHERE status = 'administered' AND child_id IN (
			SELECT id FROM children WHERE gn_division = $1
		)
	`, assignedArea).Scan(&vaccinatedCount)
	if err != nil {
		return nil, err
	}
	summary["vaccinatedCount"] = vaccinatedCount

	// Vaccination coverage percentage
	var coverage float64
	if totalChildren > 0 {
		coverage = (float64(vaccinatedCount) / float64(totalChildren)) * 100
	}
	summary["coveragePercentage"] = coverage

	// Missed vaccinations (overdue - next_due_date is past and status is not administered or cancelled)
	var missedVaccinations int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT child_id) FROM vaccination_records 
		WHERE next_due_date < CURRENT_DATE AND status NOT IN ('administered', 'cancelled')
		AND child_id IN (SELECT id FROM children WHERE gn_division = $1)
	`, assignedArea).Scan(&missedVaccinations)
	if err != nil {
		return nil, err
	}
	summary["missedVaccinations"] = missedVaccinations

	// Upcoming vaccinations (due in next 7 days and not yet administered)
	var upcomingVaccinations int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT child_id) FROM vaccination_records 
		WHERE next_due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 days'
		AND status NOT IN ('administered', 'cancelled')
		AND child_id IN (SELECT id FROM children WHERE gn_division = $1)
	`, assignedArea).Scan(&upcomingVaccinations)
	if err != nil {
		return nil, err
	}
	summary["upcomingVaccinations"] = upcomingVaccinations

	// New registrations this month
	var newRegistrations int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM children 
		WHERE gn_division = $1 
		AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)
	`, assignedArea).Scan(&newRegistrations)
	if err != nil {
		return nil, err
	}
	summary["newRegistrationsThisMonth"] = newRegistrations

	// Growth records recorded this month
	var growthRecords int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM growth_records 
		WHERE child_id IN (SELECT id FROM children WHERE gn_division = $1)
		AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)
	`, assignedArea).Scan(&growthRecords)
	if err != nil {
		return nil, err
	}
	summary["growthRecordsThisMonth"] = growthRecords

	// Scheduled clinics for this area
	var scheduledClinics int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM clinic_schedules 
		WHERE gn_division = $1 AND status = 'scheduled'
	`, assignedArea).Scan(&scheduledClinics)
	if err != nil {
		return nil, err
	}
	summary["scheduledClinics"] = scheduledClinics

	// Children by vaccination status breakdown
	// On Track: has at least one administered vaccination, no overdue items
	// Delayed: has at least one overdue (missed) vaccination or no vaccinations yet but due soon
	// Not Started: no vaccination records at all
	statusBreakdown := make(map[string]interface{})

	var onTrack, delayed, notStarted int

	// On Track: children with vaccinations and no overdue items
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT c.id) FROM children c
		WHERE c.gn_division = $1
		AND EXISTS (SELECT 1 FROM vaccination_records vr WHERE vr.child_id = c.id AND vr.status = 'administered')
		AND NOT EXISTS (SELECT 1 FROM vaccination_records vr WHERE vr.child_id = c.id AND vr.next_due_date < CURRENT_DATE AND vr.status NOT IN ('administered', 'cancelled'))
	`, assignedArea).Scan(&onTrack)
	if err != nil {
		return nil, err
	}

	// Not Started: children with no vaccination records
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT c.id) FROM children c
		WHERE c.gn_division = $1
		AND NOT EXISTS (SELECT 1 FROM vaccination_records vr WHERE vr.child_id = c.id)
	`, assignedArea).Scan(&notStarted)
	if err != nil {
		return nil, err
	}

	// Delayed: children with overdue vaccinations (totalChildren - onTrack - notStarted)
	delayed = totalChildren - onTrack - notStarted
	if delayed < 0 {
		delayed = 0
	}

	statusBreakdown["onTrack"] = onTrack
	statusBreakdown["delayed"] = delayed
	statusBreakdown["notStarted"] = notStarted
	summary["vaccinationStatusBreakdown"] = statusBreakdown

	return summary, nil
}
