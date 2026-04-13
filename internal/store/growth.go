package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ncvms/internal/growth"
	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GrowthRecordStore struct {
	pool        *pgxpool.Pool
	whoAssessor *growth.Assessor
}

type growthThreshold struct {
	AgeMonth int
	Low      float64
	High     float64
}

var weightForAgeThresholds = []growthThreshold{
	{AgeMonth: 0, Low: 2.5, High: 4.5},
	{AgeMonth: 1, Low: 3.2, High: 5.5},
	{AgeMonth: 2, Low: 4.0, High: 6.5},
	{AgeMonth: 3, Low: 4.5, High: 7.2},
	{AgeMonth: 6, Low: 6.0, High: 9.0},
	{AgeMonth: 9, Low: 7.0, High: 10.5},
	{AgeMonth: 12, Low: 7.8, High: 11.5},
	{AgeMonth: 18, Low: 8.5, High: 12.5},
	{AgeMonth: 24, Low: 9.0, High: 13.5},
	{AgeMonth: 36, Low: 11.0, High: 16.0},
	{AgeMonth: 48, Low: 12.5, High: 18.0},
	{AgeMonth: 60, Low: 14.0, High: 20.0},
}

var heightForAgeThresholds = []growthThreshold{
	{AgeMonth: 0, Low: 45, High: 55},
	{AgeMonth: 1, Low: 50, High: 58},
	{AgeMonth: 2, Low: 54, High: 62},
	{AgeMonth: 3, Low: 57, High: 65},
	{AgeMonth: 6, Low: 63, High: 72},
	{AgeMonth: 9, Low: 68, High: 76},
	{AgeMonth: 12, Low: 72, High: 80},
	{AgeMonth: 18, Low: 78, High: 86},
	{AgeMonth: 24, Low: 82, High: 92},
	{AgeMonth: 36, Low: 90, High: 100},
	{AgeMonth: 48, Low: 98, High: 108},
	{AgeMonth: 60, Low: 105, High: 115},
}

func NewGrowthRecordStore(pool *pgxpool.Pool, whoAssessor ...*growth.Assessor) *GrowthRecordStore {
	var assessor *growth.Assessor
	if len(whoAssessor) > 0 {
		assessor = whoAssessor[0]
	}
	return &GrowthRecordStore{pool: pool, whoAssessor: assessor}
}

func (s *GrowthRecordStore) Create(ctx context.Context, id, childID, recordedDate, recordedBy, notes string,
	height, weight, headCirc *float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO growth_records (id, child_id, recorded_date, height, weight, head_circumference, recorded_by, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, childID, recordedDate, height, weight, headCirc, recordedBy, notes)
	return err
}

func (s *GrowthRecordStore) ByChildID(ctx context.Context, childID, startDate, endDate string) ([]models.GrowthRecord, error) {
	q := `
		SELECT gr.id, gr.child_id, gr.recorded_date::text, gr.height, gr.weight, gr.head_circumference,
		       gr.recorded_by, gr.notes, gr.created_at, c.date_of_birth::text, c.gender
		FROM growth_records gr
		JOIN children c ON c.id = gr.child_id
		WHERE gr.child_id = $1`
	args := []interface{}{childID}
	if startDate != "" {
		q += ` AND gr.recorded_date >= $2`
		args = append(args, startDate)
	}
	if endDate != "" {
		q += fmt.Sprintf(` AND gr.recorded_date <= $%d`, len(args)+1)
		args = append(args, endDate)
	}
	q += ` ORDER BY gr.recorded_date ASC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.GrowthRecord
	for rows.Next() {
		var r models.GrowthRecord
		var dobText, sex string
		err := rows.Scan(&r.RecordId, &r.ChildId, &r.RecordedDate, &r.Height, &r.Weight, &r.HeadCircumference, &r.RecordedBy, &r.Notes, &r.CreatedAt, &dobText, &sex)
		if err != nil {
			return nil, err
		}
		ageMonths, ok := deriveAgeInMonths(dobText, r.RecordedDate)
		if ok {
			r.AgeInMonths = &ageMonths
			s.assignClassification(&r, sex, ageMonths)
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *GrowthRecordStore) ChartsByChildID(ctx context.Context, childID, startDate, endDate string) (*models.ChildGrowthCharts, error) {
	history, err := s.ByChildID(ctx, childID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	sex, _ := s.childSex(ctx, childID)
	result := &models.ChildGrowthCharts{
		ChildId:      childID,
		HistoryTable: history,
		WeightVsAge:  make([]models.GrowthChartPoint, 0, len(history)),
		HeightVsAge:  make([]models.GrowthChartPoint, 0, len(history)),
	}
	for _, record := range history {
		if record.AgeInMonths == nil {
			continue
		}
		result.WeightVsAge = append(result.WeightVsAge, models.GrowthChartPoint{
			DateOfVisit: record.RecordedDate,
			AgeInMonths: *record.AgeInMonths,
			Value:       record.Weight,
			Status:      record.WeightStatus,
			ZScore:      record.WeightZScore,
			Metric:      "weight",
		})
		result.HeightVsAge = append(result.HeightVsAge, models.GrowthChartPoint{
			DateOfVisit: record.RecordedDate,
			AgeInMonths: *record.AgeInMonths,
			Value:       record.Height,
			Status:      record.HeightStatus,
			ZScore:      record.HeightZScore,
			Metric:      "height",
		})
	}
	if s.whoAssessor != nil && s.whoAssessor.HasStandardData() {
		result.WeightReference = mapReferencePoints(s.whoAssessor.Series(growth.MetricWeightForAge, sex))
		result.HeightReference = mapReferencePoints(s.whoAssessor.Series(growth.MetricHeightForAge, sex))
		result.ReferenceVersion = s.whoAssessor.Version()
	}
	return result, nil
}

func (s *GrowthRecordStore) assignClassification(r *models.GrowthRecord, sex string, ageMonths int) {
	if s.whoAssessor != nil && s.whoAssessor.HasStandardData() {
		weightStatus, weightZ, weightOK := s.whoAssessor.Assess(growth.MetricWeightForAge, sex, ageMonths, r.Weight)
		heightStatus, heightZ, heightOK := s.whoAssessor.Assess(growth.MetricHeightForAge, sex, ageMonths, r.Height)
		if weightOK {
			r.WeightStatus = weightStatus
			r.WeightZScore = weightZ
		}
		if heightOK {
			r.HeightStatus = heightStatus
			r.HeightZScore = heightZ
		}
		if weightOK || heightOK {
			return
		}
	}
	r.WeightStatus = classifyWeightStatus(ageMonths, r.Weight)
	r.HeightStatus = classifyHeightStatus(ageMonths, r.Height)
}

func (s *GrowthRecordStore) childSex(ctx context.Context, childID string) (string, bool) {
	var sex string
	if err := s.pool.QueryRow(ctx, `SELECT gender FROM children WHERE id = $1`, childID).Scan(&sex); err != nil {
		return "", false
	}
	return sex, true
}

func mapReferencePoints(points []growth.ReferencePoint) []models.GrowthReferencePoint {
	out := make([]models.GrowthReferencePoint, 0, len(points))
	for _, p := range points {
		out = append(out, models.GrowthReferencePoint{
			AgeInMonths: p.AgeMonth,
			SDNeg3:      p.SDNeg3,
			SDNeg2:      p.SDNeg2,
			SDNeg1:      p.SDNeg1,
			Median:      p.Median,
			SDPos1:      p.SDPos1,
			SDPos2:      p.SDPos2,
			SDPos3:      p.SDPos3,
		})
	}
	return out
}

func deriveAgeInMonths(dateOfBirth, visitDate string) (int, bool) {
	dob, ok := parseDate(dateOfBirth)
	if !ok {
		return 0, false
	}
	visit, ok := parseDate(visitDate)
	if !ok {
		return 0, false
	}
	if visit.Before(dob) {
		return 0, true
	}
	years := visit.Year() - dob.Year()
	months := int(visit.Month()) - int(dob.Month())
	totalMonths := years*12 + months
	if visit.Day() < dob.Day() {
		totalMonths--
	}
	if totalMonths < 0 {
		totalMonths = 0
	}
	return totalMonths, true
}

func parseDate(value string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05-07:00",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func classifyWeightStatus(ageInMonths int, weight *float64) string {
	if weight == nil {
		return ""
	}
	threshold := matchThreshold(ageInMonths, weightForAgeThresholds)
	if *weight < threshold.Low {
		return "underweight"
	}
	if *weight > threshold.High {
		return "overweight"
	}
	return "normal"
}

func classifyHeightStatus(ageInMonths int, height *float64) string {
	if height == nil {
		return ""
	}
	threshold := matchThreshold(ageInMonths, heightForAgeThresholds)
	if *height < threshold.Low {
		return "stunted"
	}
	return "normal"
}

func matchThreshold(ageInMonths int, thresholds []growthThreshold) growthThreshold {
	selected := thresholds[0]
	for _, t := range thresholds {
		if t.AgeMonth <= ageInMonths {
			selected = t
			continue
		}
		break
	}
	return selected
}

func (s *GrowthRecordStore) HasWHOReference() bool {
	return s.whoAssessor != nil && s.whoAssessor.HasStandardData()
}

func (s *GrowthRecordStore) WHOPayloadByChildID(ctx context.Context, childID, startDate, endDate string) (*models.ChildWHOGrowthPayload, error) {
	history, err := s.ByChildID(ctx, childID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	sex, ok := s.childSex(ctx, childID)
	if !ok {
		return nil, nil
	}
	payload := &models.ChildWHOGrowthPayload{
		Version:      "prototype-threshold-fallback",
		Metadata:     map[string]string{},
		ChildID:      childID,
		Sex:          strings.ToLower(sex),
		Indicators:   map[string][]models.GrowthReferencePoint{},
		Observations: make([]models.ChildGrowthWHOObservation, 0, len(history)),
	}

	for _, r := range history {
		if r.AgeInMonths == nil {
			continue
		}
		payload.Observations = append(payload.Observations, models.ChildGrowthWHOObservation{
			DateOfVisit:  r.RecordedDate,
			AgeMonth:     *r.AgeInMonths,
			Weight:       r.Weight,
			Height:       r.Height,
			WeightStatus: r.WeightStatus,
			HeightStatus: r.HeightStatus,
			WeightZScore: r.WeightZScore,
			HeightZScore: r.HeightZScore,
		})
	}

	if s.HasWHOReference() {
		payload.Version = s.whoAssessor.Version()
		payload.Metadata = s.whoAssessor.Metadata()
		payload.Indicators[growth.MetricWeightForAge] = mapReferencePoints(s.whoAssessor.Series(growth.MetricWeightForAge, sex))
		payload.Indicators[growth.MetricHeightForAge] = mapReferencePoints(s.whoAssessor.Series(growth.MetricHeightForAge, sex))
		return payload, nil
	}

	payload.Metadata["note"] = "WHO reference data file is not loaded; using simplified fallback thresholds"
	payload.Indicators[growth.MetricWeightForAge] = thresholdToReference(weightForAgeThresholds)
	payload.Indicators[growth.MetricHeightForAge] = thresholdToReference(heightForAgeThresholds)
	return payload, nil
}

func thresholdToReference(thresholds []growthThreshold) []models.GrowthReferencePoint {
	out := make([]models.GrowthReferencePoint, 0, len(thresholds))
	for _, t := range thresholds {
		out = append(out, models.GrowthReferencePoint{
			AgeInMonths: t.AgeMonth,
			SDNeg2:      t.Low,
			Median:      (t.Low + t.High) / 2,
			SDPos2:      t.High,
		})
	}
	return out
}
