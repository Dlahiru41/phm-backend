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
