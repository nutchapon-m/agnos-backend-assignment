package hospitaldb

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
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

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (hospitalbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, h hospitalbus.Hospital) (int, error) {
	hospital := toDBHospital(h)
	args := []any{
		hospital.Code,
		hospital.Name,
		hospital.ProvinceCode,
		hospital.IsActive,
		hospital.CreatedAt,
		hospital.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO hospitals(id, code, name, province_code, is_active, created_at, updated_at)
	VALUES (nextval('hospitals_id_seq'), $1, $2, $3, $4, $5, $6)
	RETURNING id;
	`
	if err := s.db.QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		if isUniqueViolation(err) {
			return 0, sqldb.ErrDBDuplicatedEntry
		}
		return id, err
	}
	return id, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (hospitalbus.Hospital, error) {
	query := `
	SELECT id, code, name, province_code, is_active, created_at, updated_at, deleted_at
	FROM hospitals
	WHERE id = $1 AND deleted_at IS NULL`

	var hsp hospital
	if err := sqlx.GetContext(ctx, s.db, &hsp, query, id); err != nil {
		return hospitalbus.Hospital{}, err
	}
	return toHospital(hsp), nil
}

func (s *Store) Query(ctx context.Context, filter hospitalbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalbus.Hospital, error) {
	data := map[string]any{}
	if !pg.IsZero() {
		data["offset"] = (pg.Number() - 1) * pg.RowsPerPage()
		data["rows_per_page"] = pg.RowsPerPage()
	}

	query := `SELECT * FROM hospitals`

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

	var hospitals []hospital
	if err := sqlx.SelectContext(ctx, s.db, &hospitals, query, args...); err != nil {
		return nil, err
	}

	hospitalsbus := make([]hospitalbus.Hospital, len(hospitals))
	for i, val := range hospitals {
		hospitalsbus[i] = toHospital(val)
	}
	return hospitalsbus, nil
}

func (s *Store) Update(ctx context.Context, h hospitalbus.Hospital) error {
	hospital := toDBHospital(h)
	query := `
	UPDATE hospitals SET
		code = :code,
		name = :name,
		province_code = :province_code,
		is_active = :is_active,
		updated_at = :updated_at
	WHERE id = :id AND deleted_at IS NULL;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, hospital); err != nil {
		if isUniqueViolation(err) {
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
		"UPDATE hospitals SET deleted_at = :deleted_at WHERE id = :id",
	}

	for _, query := range queies {
		if _, err := sqlx.NamedExecContext(ctx, s.db, query, data); err != nil {
			return err
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}
