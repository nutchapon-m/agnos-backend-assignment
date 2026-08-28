package sqldb

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	undefinedTable      = "42P01"
)

var (
	ErrDBNotFound            = sql.ErrNoRows
	ErrDBDuplicatedEntry     = errors.New("duplicated entry")
	ErrDBForeignKeyViolation = errors.New("referenced row does not exist")
	ErrUndefinedTable        = errors.New("undefined table")
)

type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	Schema       string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")
	if cfg.Schema != "" {
		q.Set("search_path", cfg.Schema)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

func StatusCheck(ctx context.Context, db *sqlx.DB) error {

	var cancel context.CancelFunc
	// If the user doesn't give us a deadline set 1 second.
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	}
	defer cancel()

	for attempts := 1; ; attempts++ {
		if err := db.Ping(); err == nil {
			break
		}

		time.Sleep(time.Duration(attempts) * 1 * time.Second)

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Run a simple query to determine connectivity.
	// Running this query forces a round trip through the database.
	const q = `SELECT TRUE`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}

func PrepareStmtExt(ctx context.Context, ext sqlx.ExtContext, query string) (*sqlx.Stmt, error) {
	switch e := ext.(type) {
	case *sqlx.DB:
		return e.PreparexContext(ctx, query)
	case *sqlx.Tx:
		return e.PreparexContext(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported ExtContext type")
	}
}

// ExecContext is a helper function to execute a CUD operation with
// logging and tracing.
func ExecContext(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string) error {
	return NamedExecContext(ctx, log, db, query, struct{}{})
}

// NamedExecContext is a helper function to execute a CUD operation with
// logging and tracing where field replacement is necessary.
func NamedExecContext(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any) (err error) {
	q := queryString(query, data)

	defer func() {
		if err != nil {
			switch data.(type) {
			case struct{}:
				log.Infoc(ctx, 6, "database.NamedExecContext", "query", q, "ERROR", err)
			default:
				log.Infoc(ctx, 5, "database.NamedExecContext", "query", q, "ERROR", err)
			}
		}
	}()

	if _, err := sqlx.NamedExecContext(ctx, db, query, data); err != nil {
		var pqerr *pgconn.PgError
		if errors.As(err, &pqerr) {
			switch pqerr.Code {
			case undefinedTable:
				return ErrUndefinedTable
			case uniqueViolation:
				return ErrDBDuplicatedEntry
			case foreignKeyViolation:
				return ErrDBForeignKeyViolation
			}
		}
		return err
	}

	return nil
}

// QuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshalled into a slice.
func QuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, struct{}{}, dest, false)
}

// NamedQuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshalled into a slice where field replacement is
// necessary.
func NamedQuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, data, dest, false)
}

// NamedQuerySliceUsingIn is a helper function for executing queries that return
// a collection of data to be unmarshalled into a slice where field replacement
// is necessary. Use this if the query has an IN clause.
func NamedQuerySliceUsingIn[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, data, dest, true)
}

func namedQuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T, withIn bool) (err error) {
	q := queryString(query, data)

	defer func() {
		if err != nil {
			log.Infoc(ctx, 6, "database.NamedQuerySlice", "query", q, "ERROR", err)
		}
	}()

	var rows *sqlx.Rows

	switch withIn {
	case true:
		rows, err = func() (*sqlx.Rows, error) {
			named, args, err := sqlx.Named(query, data)
			if err != nil {
				return nil, err
			}

			query, args, err := sqlx.In(named, args...)
			if err != nil {
				return nil, err
			}

			query = db.Rebind(query)
			return db.QueryxContext(ctx, query, args...)
		}()

	default:
		rows, err = sqlx.NamedQueryContext(ctx, db, query, data)
	}

	if err != nil {
		var pqerr *pgconn.PgError
		if errors.As(err, &pqerr) && pqerr.Code == undefinedTable {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	var slice []T
	for rows.Next() {
		v := new(T)
		if err := rows.StructScan(v); err != nil {
			return err
		}
		slice = append(slice, *v)
	}
	*dest = slice

	return nil
}

// QueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type where field replacement is necessary.
func QueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, dest any) error {
	return namedQueryStruct(ctx, log, db, query, struct{}{}, dest, false)
}

// NamedQueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type where field replacement is necessary.
func NamedQueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any) error {
	return namedQueryStruct(ctx, log, db, query, data, dest, false)
}

// NamedQueryStructUsingIn is a helper function for executing queries that return
// a single value to be unmarshalled into a struct type where field replacement
// is necessary. Use this if the query has an IN clause.
func NamedQueryStructUsingIn(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any) error {
	return namedQueryStruct(ctx, log, db, query, data, dest, true)
}

func namedQueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any, withIn bool) (err error) {
	q := queryString(query, data)

	defer func() {
		if err != nil {
			log.Infoc(ctx, 6, "database.NamedQuerySlice", "query", q, "ERROR", err)
		}
	}()

	var rows *sqlx.Rows

	switch withIn {
	case true:
		rows, err = func() (*sqlx.Rows, error) {
			named, args, err := sqlx.Named(query, data)
			if err != nil {
				return nil, err
			}

			query, args, err := sqlx.In(named, args...)
			if err != nil {
				return nil, err
			}

			query = db.Rebind(query)
			return db.QueryxContext(ctx, query, args...)
		}()

	default:
		rows, err = sqlx.NamedQueryContext(ctx, db, query, data)
	}

	if err != nil {
		var pqerr *pgconn.PgError
		if errors.As(err, &pqerr) && pqerr.Code == undefinedTable {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrDBNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return err
	}

	return nil
}

// queryString provides a pretty print version of the query and parameters.
func queryString(query string, args any) string {
	query, params, err := sqlx.Named(query, args)
	if err != nil {
		return err.Error()
	}

	for _, param := range params {
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("'%s'", v)
		case []byte:
			value = fmt.Sprintf("'%s'", string(v))
		default:
			value = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, "?", value, 1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.Trim(query, " ")
}

func MaptoQuery(db sqlx.ExtContext, buff *bytes.Buffer, data map[string]any) (string, []any, error) {
	rawQuery := buff.String()
	nQuery, nArgs, err := sqlx.Named(rawQuery, data)
	if err != nil {
		return "", nil, err
	}

	if strings.Contains(nQuery, "IN") {
		nQuery, nArgs, err = sqlx.In(nQuery, nArgs...)
		if err != nil {
			return "", nil, err
		}
	}

	nQuery = db.Rebind(nQuery)
	return nQuery, nArgs, nil
}
