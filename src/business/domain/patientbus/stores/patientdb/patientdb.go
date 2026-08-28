package patientdb

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
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

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (patientbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, p patientbus.Patient) (int, error) {
	patient := toDBPatient(p)
	args := []any{
		patient.NationalID,
		patient.PassportNo,
		patient.FirstNameTH,
		patient.MiddleNameTH,
		patient.LastNameTH,
		patient.FirstNameEN,
		patient.MiddleNameEN,
		patient.LastNameEN,
		patient.DateOfBirth,
		patient.Gender,
		patient.Phone,
		patient.CreatedAt,
		patient.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO patients(id, national_id, passport_no, first_name_th, middle_name_th, last_name_th,
		first_name_en, middle_name_en, last_name_en, date_of_birth, gender, phone, created_at, updated_at)
	VALUES (nextval('patients_id_seq'), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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

func (s *Store) GetByID(ctx context.Context, id int) (patientbus.Patient, error) {
	query := `
	SELECT id, national_id, passport_no, first_name_th, middle_name_th, last_name_th,
		first_name_en, middle_name_en, last_name_en, date_of_birth, gender, phone,
		created_at, updated_at, deleted_at
	FROM patients
	WHERE id = $1 AND deleted_at IS NULL`

	var pt patient
	if err := sqlx.GetContext(ctx, s.db, &pt, query, id); err != nil {
		return patientbus.Patient{}, err
	}
	return toPatient(pt), nil
}

func (s *Store) Query(ctx context.Context, filter patientbus.QueryFilter, pg page.Page, orderBy order.By) ([]patientbus.Patient, error) {
	data := map[string]any{}
	if !pg.IsZero() {
		data["offset"] = (pg.Number() - 1) * pg.RowsPerPage()
		data["rows_per_page"] = pg.RowsPerPage()
	}

	query := `SELECT * FROM patients`

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

	var patients []patient
	if err := sqlx.SelectContext(ctx, s.db, &patients, query, args...); err != nil {
		return nil, err
	}

	patientsbus := make([]patientbus.Patient, len(patients))
	for i, val := range patients {
		patientsbus[i] = toPatient(val)
	}
	return patientsbus, nil
}

func (s *Store) Update(ctx context.Context, p patientbus.Patient) error {
	patient := toDBPatient(p)
	query := `
	UPDATE patients SET
		national_id = :national_id,
		passport_no = :passport_no,
		first_name_th = :first_name_th,
		middle_name_th = :middle_name_th,
		last_name_th = :last_name_th,
		first_name_en = :first_name_en,
		middle_name_en = :middle_name_en,
		last_name_en = :last_name_en,
		date_of_birth = :date_of_birth,
		gender = :gender,
		phone = :phone,
		updated_at = :updated_at
	WHERE id = :id AND deleted_at IS NULL;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, patient); err != nil {
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
		"UPDATE patients SET deleted_at = :deleted_at WHERE id = :id",
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
