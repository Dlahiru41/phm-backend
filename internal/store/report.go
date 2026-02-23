package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportStore struct {
	pool *pgxpool.Pool
}

func NewReportStore(pool *pgxpool.Pool) *ReportStore { return &ReportStore{pool: pool} }

func (s *ReportStore) Create(ctx context.Context, id, reportType, generatedBy, startDate, endDate, format, filePath string, filterParams map[string]interface{}) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO reports (id, report_type, generated_by, start_date, end_date, format, file_path, filter_params)
		VALUES ($1, $2, $3, NULLIF($4,'')::date, NULLIF($5,'')::date, $6, $7, $8)
	`, id, reportType, generatedBy, startDate, endDate, format, filePath, filterParams)
	return err
}

func (s *ReportStore) List(ctx context.Context, generatedBy, reportType string, page, limit int) ([]models.Report, error) {
	q := `SELECT id, report_type, generated_date, start_date::text, end_date::text FROM reports WHERE generated_by = $1`
	args := []interface{}{generatedBy}
	if reportType != "" {
		q += ` AND report_type = $2`
		args = append(args, reportType)
	}
	n := len(args)
	q += fmt.Sprintf(` ORDER BY generated_date DESC LIMIT $%d OFFSET $%d`, n+1, n+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Report
	for rows.Next() {
		var r models.Report
		var start, end *string
		err := rows.Scan(&r.ReportId, &r.ReportType, &r.GeneratedDate, &start, &end)
		if err != nil {
			return nil, err
		}
		if start != nil {
			r.StartDate = *start
		}
		if end != nil {
			r.EndDate = *end
		}
		r.DownloadUrl = "/api/v1/reports/" + r.ReportId + "/download"
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *ReportStore) GetByID(ctx context.Context, reportID string) (*models.Report, error) {
	var r models.Report
	var start, end *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, report_type, generated_by, generated_date, start_date::text, end_date::text FROM reports WHERE id = $1
	`, reportID).Scan(&r.ReportId, &r.ReportType, &r.GeneratedBy, &r.GeneratedDate, &start, &end)
	if err != nil {
		return nil, err
	}
	if start != nil {
		r.StartDate = *start
	}
	if end != nil {
		r.EndDate = *end
	}
	r.DownloadUrl = "/api/v1/reports/" + r.ReportId + "/download"
	return &r, nil
}

func (s *ReportStore) GetFilePath(ctx context.Context, reportID string) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx, `SELECT file_path FROM reports WHERE id = $1`, reportID).Scan(&path)
	return path, err
}
