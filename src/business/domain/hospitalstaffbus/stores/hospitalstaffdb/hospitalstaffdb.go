package hospitalstaffdb

import (
	"bytes"
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

type Store struct {
	db sqlx.ExtContext
}

func NewStore(db sqlx.ExtContext) *Store {
	return &Store{db: db}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (hospitalstaffbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, hs hospitalstaffbus.HospitalStaff) (int, error) {
	hospitalStaff := toDBHospitalStaff(hs)
	args := []any{
		hospitalStaff.HospitalID,
		hospitalStaff.StaffID,
		hospitalStaff.Role,
		hospitalStaff.IsPrimary,
		hospitalStaff.EffectiveFrom,
		hospitalStaff.EffectiveTo,
		hospitalStaff.CreatedAt,
		hospitalStaff.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO hospital_staffs(id, hospital_id, staff_id, role, is_primary, effective_from, effective_to, created_at, updated_at)
	VALUES (nextval('hospital_staffs_id_seq'), $1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id;
	`
	if err := s.db.QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return 0, mapped
		}
		return id, err
	}
	return id, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (hospitalstaffbus.HospitalStaff, error) {
	query := `
	SELECT id, hospital_id, staff_id, role, is_primary, effective_from, effective_to, created_at, updated_at
	FROM hospital_staffs
	WHERE id = $1`

	var hs hospitalStaff
	if err := sqlx.GetContext(ctx, s.db, &hs, query, id); err != nil {
		return hospitalstaffbus.HospitalStaff{}, err
	}
	return toHospitalStaff(hs), nil
}

func (s *Store) Query(ctx context.Context, filter hospitalstaffbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalstaffbus.HospitalStaff, error) {
	data := map[string]any{}
	if !pg.IsZero() {
		data["offset"] = (pg.Number() - 1) * pg.RowsPerPage()
		data["rows_per_page"] = pg.RowsPerPage()
	}

	query := `SELECT * FROM hospital_staffs`

	buf := bytes.NewBufferString(query)
	s.applyFilters(filter, data, buf)

	orderByClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(orderByClause)
	if !pg.IsZero() {
		buf.WriteString(" OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY")
	}

	query, args, err := sqldb.MaptoQuery(s.db, buf, data)
	if err != nil {
		return nil, err
	}

	var hospitalStaffs []hospitalStaff
	if err := sqlx.SelectContext(ctx, s.db, &hospitalStaffs, query, args...); err != nil {
		return nil, err
	}

	hospitalStaffsbus := make([]hospitalstaffbus.HospitalStaff, len(hospitalStaffs))
	for i, val := range hospitalStaffs {
		hospitalStaffsbus[i] = toHospitalStaff(val)
	}
	return hospitalStaffsbus, nil
}

func (s *Store) Update(ctx context.Context, hs hospitalstaffbus.HospitalStaff) error {
	hospitalStaff := toDBHospitalStaff(hs)
	query := `
	UPDATE hospital_staffs SET
		role = :role,
		is_primary = :is_primary,
		effective_from = :effective_from,
		effective_to = :effective_to,
		updated_at = :updated_at
	WHERE id = :id;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, hospitalStaff); err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return mapped
		}
		return err
	}
	return nil
}

// Delete removes the row. The table carries no deleted_at column, so there is
// nothing to soft delete.
func (s *Store) Delete(ctx context.Context, id int) error {
	data := map[string]any{
		"id": id,
	}
	queies := []string{
		"DELETE FROM hospital_staffs WHERE id = :id",
	}

	for _, query := range queies {
		if _, err := sqlx.NamedExecContext(ctx, s.db, query, data); err != nil {
			return err
		}
	}
	return nil
}

// mapConstraintError translates postgres constraint codes into store level
// errors. It returns nil when err is not a constraint violation.
func mapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}

	switch pgErr.Code {
	case uniqueViolation:
		return sqldb.ErrDBDuplicatedEntry
	case foreignKeyViolation:
		return sqldb.ErrDBForeignKeyViolation
	}
	return nil
}
