package staffdb

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

const uniqueViolation = "23505"

type Store struct {
	db sqlx.ExtContext
}

func NewStore(db sqlx.ExtContext) *Store {
	return &Store{db: db}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (staffbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, st staffbus.Staff) (int, error) {
	staff := toDBStaff(st)
	args := []any{
		staff.UserID,
		staff.EmployeeCode,
		staff.FirstName,
		staff.LastName,
		staff.Email,
		staff.LicenseNo,
		staff.IsActive,
		staff.CreatedAt,
		staff.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO staffs(id, user_id, employee_code, first_name, last_name, email, license_no, is_active, created_at, updated_at)
	VALUES (nextval('staffs_id_seq'), $1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING id;
	`
	if err := s.db.QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return 0, sqldb.ErrDBDuplicatedEntry
		}
		return id, err
	}
	return id, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (staffbus.Staff, error) {
	query := `
	SELECT id, user_id, employee_code, first_name, last_name, email, license_no, is_active, created_at, updated_at, deleted_at
	FROM staffs
	WHERE id = $1 AND deleted_at IS NULL`

	var stf staff
	if err := sqlx.GetContext(ctx, s.db, &stf, query, id); err != nil {
		return staffbus.Staff{}, err
	}
	return toStaff(stf), nil
}

func (s *Store) Query(ctx context.Context, filter staffbus.QueryFilter, p page.Page, orderBy order.By) ([]staffbus.Staff, error) {
	data := map[string]any{}
	if !p.IsZero() {
		data["offset"] = (p.Number() - 1) * p.RowsPerPage()
		data["rows_per_page"] = p.RowsPerPage()
	}

	query := `SELECT * FROM staffs`

	buf := bytes.NewBufferString(query)
	s.applyFilters(filter, data, buf)

	orderByClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(orderByClause)
	if !p.IsZero() {
		buf.WriteString(" OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY")
	}

	query, args, err := sqldb.MaptoQuery(s.db, buf, data)
	if err != nil {
		return nil, err
	}

	var staffs []staff
	if err := sqlx.SelectContext(ctx, s.db, &staffs, query, args...); err != nil {
		return nil, err
	}

	staffsbus := make([]staffbus.Staff, len(staffs))
	for i, val := range staffs {
		staffsbus[i] = toStaff(val)
	}
	return staffsbus, nil
}

func (s *Store) Update(ctx context.Context, st staffbus.Staff) error {
	staff := toDBStaff(st)
	query := `
	UPDATE staffs SET
		employee_code = :employee_code,
		first_name = :first_name,
		last_name = :last_name,
		email = :email,
		license_no = :license_no,
		is_active = :is_active,
		updated_at = :updated_at
	WHERE id = :id AND deleted_at IS NULL;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, staff); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return sqldb.ErrDBDuplicatedEntry
		}
		return err
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id int) error {
	data := map[string]any{
		"id":         id,
		"deleted_at": time.Now(),
	}
	queies := []string{
		"UPDATE staffs SET deleted_at = :deleted_at WHERE id = :id",
	}

	for _, query := range queies {
		if _, err := sqlx.NamedExecContext(ctx, s.db, query, data); err != nil {
			return err
		}
	}
	return nil
}
