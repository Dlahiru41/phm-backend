package store

import (
	"context"
	"fmt"

	"ncvms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChildStore struct {
	pool *pgxpool.Pool
}

func NewChildStore(pool *pgxpool.Pool) *ChildStore { return &ChildStore{pool: pool} }

func (s *ChildStore) Create(ctx context.Context, id, regNum, firstName, lastName, dob, gender, bloodGroup string,
	birthWeight, birthHeight, headCirc *float64, motherName, motherNIC, fatherName, fatherNIC string,
	registeredBy, district, dsDiv, gnDiv, address, areaCode string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO children (id, registration_number, first_name, last_name, date_of_birth, gender, blood_group,
			birth_weight, birth_height, head_circumference, mother_name, mother_nic, father_name, father_nic,
			registered_by, district, ds_division, gn_division, address, area_code)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, id, regNum, firstName, lastName, dob, gender, bloodGroup, birthWeight, birthHeight, headCirc,
		motherName, motherNIC, fatherName, fatherNIC, registeredBy, district, dsDiv, gnDiv, address, areaCode)
	return err
}

func (s *ChildStore) GetByID(ctx context.Context, childID string) (*models.ChildDetail, error) {
	var c models.ChildDetail
	var bw, bh, hc *float64
	var parentID, regBy *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, registration_number, first_name, last_name, date_of_birth, gender, blood_group,
		       birth_weight, birth_height, head_circumference, parent_id, registered_by, area_code,
		       COALESCE(area_code,'') as area_name, mother_name, mother_nic, father_name, father_nic,
		       district, ds_division, gn_division, address, created_at
		FROM children WHERE id = $1
	`, childID).Scan(&c.ChildId, &c.RegistrationNumber, &c.FirstName, &c.LastName, &c.DateOfBirth, &c.Gender, &c.BloodGroup,
		&bw, &bh, &hc, &parentID, &regBy, &c.AreaCode, &c.AreaName, &c.MotherName, &c.MotherNIC, &c.FatherName, &c.FatherNIC,
		&c.District, &c.DsDivision, &c.GnDivision, &c.Address, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.BirthWeight = bw
	c.BirthHeight = bh
	c.HeadCircumference = hc
	c.ParentId = parentID
	c.RegisteredBy = regBy
	return &c, nil
}

func (s *ChildStore) GetByRegistrationNumber(ctx context.Context, regNum string) (*models.Child, error) {
	var c models.Child
	var bw, bh, hc *float64
	var parentID, regBy *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, registration_number, first_name, last_name, date_of_birth, gender, blood_group,
		       birth_weight, birth_height, head_circumference, parent_id, registered_by, area_code,
		       COALESCE(area_code,'') as area_name, created_at
		FROM children WHERE registration_number = $1
	`, regNum).Scan(&c.ChildId, &c.RegistrationNumber, &c.FirstName, &c.LastName, &c.DateOfBirth, &c.Gender, &c.BloodGroup,
		&bw, &bh, &hc, &parentID, &regBy, &c.AreaCode, &c.AreaName, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.BirthWeight = bw
	c.BirthHeight = bh
	c.HeadCircumference = hc
	c.ParentId = parentID
	c.RegisteredBy = regBy
	return &c, nil
}

func (s *ChildStore) LinkParent(ctx context.Context, childID, parentID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE children SET parent_id = $2 WHERE id = $1`, childID, parentID)
	return err
}

func (s *ChildStore) ByParentID(ctx context.Context, parentID string) ([]models.Child, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, registration_number, first_name, last_name, date_of_birth, gender, blood_group,
		       birth_weight, birth_height, head_circumference, parent_id, registered_by, area_code,
		       COALESCE(area_code,'') as area_name, created_at
		FROM children WHERE parent_id = $1 ORDER BY created_at DESC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChildren(rows)
}

func (s *ChildStore) ByPHMID(ctx context.Context, phmID string) ([]models.Child, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.registration_number, c.first_name, c.last_name, c.date_of_birth, c.gender, c.blood_group,
		       c.birth_weight, c.birth_height, c.head_circumference, c.parent_id, c.registered_by, c.area_code,
		       COALESCE(c.area_code,'') as area_name, c.created_at
		FROM children c
		JOIN users u ON u.id = $1 AND u.area_code = c.area_code
		ORDER BY c.created_at DESC
	`, phmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChildren(rows)
}

func (s *ChildStore) ListMOH(ctx context.Context, areaCode, status, search string, page, limit int) (total int, list []models.Child, err error) {
	base := `FROM children c WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if areaCode != "" {
		base += fmt.Sprintf(` AND c.area_code = $%d`, idx)
		args = append(args, areaCode)
		idx++
	}
	if search != "" {
		base += fmt.Sprintf(` AND (c.first_name ILIKE $%d OR c.last_name ILIKE $%d OR c.registration_number ILIKE $%d)`, idx, idx+1, idx+2)
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		idx += 3
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) `+base, args...).Scan(&total)
	if err != nil {
		return 0, nil, err
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.registration_number, c.first_name, c.last_name, c.date_of_birth, c.gender, c.blood_group,
		       c.birth_weight, c.birth_height, c.head_circumference, c.parent_id, c.registered_by, c.area_code,
		       COALESCE(c.area_code,'') as area_name, c.created_at
		`+base+` ORDER BY c.created_at DESC LIMIT $`+fmt.Sprint(idx)+` OFFSET $`+fmt.Sprint(idx+1), args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	list, err = scanChildren(rows)
	if err != nil {
		return 0, nil, err
	}
	_ = status
	return total, list, nil
}

func (s *ChildStore) Update(ctx context.Context, childID string, firstName, lastName, bloodGroup, address *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE children SET first_name = COALESCE($2, first_name), last_name = COALESCE($3, last_name),
			blood_group = COALESCE($4, blood_group), address = COALESCE($5, address) WHERE id = $1
	`, childID, firstName, lastName, bloodGroup, address)
	return err
}

func scanChildren(rows pgx.Rows) ([]models.Child, error) {
	var list []models.Child
	for rows.Next() {
		var c models.Child
		var bw, bh, hc *float64
		var parentID, regBy *string
		err := rows.Scan(&c.ChildId, &c.RegistrationNumber, &c.FirstName, &c.LastName, &c.DateOfBirth, &c.Gender, &c.BloodGroup,
			&bw, &bh, &hc, &parentID, &regBy, &c.AreaCode, &c.AreaName, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		c.BirthWeight = bw
		c.BirthHeight = bh
		c.HeadCircumference = hc
		c.ParentId = parentID
		c.RegisteredBy = regBy
		list = append(list, c)
	}
	return list, rows.Err()
}
