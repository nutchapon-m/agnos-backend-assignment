package hospitalpatientdb

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
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

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (hospitalpatientbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, hp hospitalpatientbus.HospitalPatient) (int, error) {
	hospitalPatient := toDBHospitalPatient(hp)
	args := []any{
		hospitalPatient.HospitalID,
		hospitalPatient.PatientID,
		hospitalPatient.HN,
		hospitalPatient.Status,
		hospitalPatient.RegisteredAt,
		hospitalPatient.CreatedAt,
		hospitalPatient.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO hospital_patients(id, hospital_id, patient_id, hn, status, registered_at, created_at, updated_at)
	VALUES (nextval('hospital_patients_id_seq'), $1, $2, $3, $4, $5, $6, $7)
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

func (s *Store) GetByID(ctx context.Context, id int) (hospitalpatientbus.HospitalPatient, error) {
	query := `
	SELECT id, hospital_id, patient_id, hn, status, registered_at, created_at, updated_at, deleted_at
	FROM hospital_patients
	WHERE id = $1 AND deleted_at IS NULL`

	var hp hospitalPatient
	if err := sqlx.GetContext(ctx, s.db, &hp, query, id); err != nil {
		return hospitalpatientbus.HospitalPatient{}, err
	}
	return toHospitalPatient(hp), nil
}

func (s *Store) Query(ctx context.Context, filter hospitalpatientbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalpatientbus.HospitalPatient, error) {
	data := map[string]any{}
	if !pg.IsZero() {
		data["offset"] = (pg.Number() - 1) * pg.RowsPerPage()
		data["rows_per_page"] = pg.RowsPerPage()
	}

	query := `SELECT * FROM hospital_patients`

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

	var hospitalPatients []hospitalPatient
	if err := sqlx.SelectContext(ctx, s.db, &hospitalPatients, query, args...); err != nil {
		return nil, err
	}

	hospitalPatientsbus := make([]hospitalpatientbus.HospitalPatient, len(hospitalPatients))
	for i, val := range hospitalPatients {
		hospitalPatientsbus[i] = toHospitalPatient(val)
	}
	return hospitalPatientsbus, nil
}

func (s *Store) Update(ctx context.Context, hp hospitalpatientbus.HospitalPatient) error {
	hospitalPatient := toDBHospitalPatient(hp)
	query := `
	UPDATE hospital_patients SET
		hn = :hn,
		status = :status,
		registered_at = :registered_at,
		updated_at = :updated_at
	WHERE id = :id AND deleted_at IS NULL;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, hospitalPatient); err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return mapped
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
		"UPDATE hospital_patients SET deleted_at = :deleted_at WHERE id = :id",
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
