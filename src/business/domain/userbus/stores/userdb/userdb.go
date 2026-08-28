package userdb

import (
	"bytes"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type Store struct {
	db sqlx.ExtContext
}

func NewStore(db sqlx.ExtContext) *Store {
	return &Store{db: db}
}

func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (userbus.Store, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		db: ec,
	}

	return &store, nil
}

func (s *Store) Create(ctx context.Context, u userbus.User) (int, error) {
	user := toDBUser(u)
	args := []any{
		user.Username,
		user.Password,
		user.CreatedAt,
		user.UpdatedAt,
	}

	var id int
	query := `
	INSERT INTO users(username, password, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id;
	`
	if err := s.db.QueryRowxContext(ctx, query, args...).Scan(&id); err != nil {
		return id, err
	}
	return id, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (userbus.User, error) {
	query := `
	SELECT id, username, password, created_at, updated_at, deleted_at 
	FROM users
	WHERE id = $1 AND deleted_at IS NULL`

	var usr user
	err := sqlx.GetContext(ctx, s.db, &usr, query, id)
	if err != nil {
		return userbus.User{}, err
	}
	return toUser(usr), nil

}

func (s *Store) Query(ctx context.Context, filter userbus.QueryFilter, p page.Page, orderBy order.By) ([]userbus.User, error) {
	data := map[string]any{}
	if !p.IsZero() {
		data["offset"] = (p.Number() - 1) * p.RowsPerPage()
		data["rows_per_page"] = p.RowsPerPage()
	}

	query := `SELECT * FROM users`

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

	var users []user
	if err := sqlx.SelectContext(ctx, s.db, &users, query, args...); err != nil {
		return nil, err
	}

	usersbus := make([]userbus.User, len(users))
	for i, val := range users {
		usersbus[i] = toUser(val)
	}
	return usersbus, nil
}

func (s *Store) Update(ctx context.Context, u userbus.User) error {
	user := toDBUser(u)
	query := `
	UPDATE users SET
		username = :username,
		updated_at = :updated_at,
	WHERE id = :id;
	`
	if _, err := sqlx.NamedExecContext(ctx, s.db, query, user); err != nil {
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
		"UPDATE users SET deleted_at = :deleted_at WHERE id = :id",
	}

	for _, query := range queies {
		if _, err := sqlx.NamedExecContext(ctx, s.db, query, data); err != nil {
			return err
		}
	}
	return nil
}
