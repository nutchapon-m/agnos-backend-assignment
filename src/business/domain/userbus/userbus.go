package userbus

import (
	"context"
	"errors"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

var (
	ErrNotFound   = errors.New("error user not found")
	ErrDuplicate  = errors.New("error user already exist")
	ErrUnexpected = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, u User) (int, error)
	GetByID(ctx context.Context, id int) (User, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error)
	Update(ctx context.Context, u User) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, nu NewUser) (User, error)
	GetByID(ctx context.Context, id int) (User, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error)
	Delete(ctx context.Context, id int) error
}

type business struct {
	log   *logger.Logger
	store Store
}

func NewBusiness(log *logger.Logger, store Store) *business {
	return &business{log: log, store: store}
}

func (bus *business) NewWithTx(tx sqldb.CommitRollbacker) (Business, error) {
	storer, err := bus.store.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	business := NewBusiness(bus.log, storer)
	return business, nil
}

func (bus *business) Create(ctx context.Context, nu NewUser) (User, error) {
	now := time.Now()
	user := User{
		Username:  nu.Username,
		Password:  nu.Password,
		CreatedAt: now,
		UpdatedAt: now,
	}

	id, err := bus.store.Create(ctx, user)
	if err != nil {
		bus.log.Error(ctx, "create user error", "err", err)
		return User{}, err
	}
	user.ID = id
	return user, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (User, error) {
	user, err := bus.store.GetByID(ctx, id)
	if err != nil {
		if err == sqldb.ErrDBNotFound {
			bus.log.Error(ctx, "error get user", "err", err)
			return User{}, ErrNotFound
		}
		bus.log.Error(ctx, "error get user", "err", err)
		return User{}, ErrUnexpected
	}
	return user, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error) {
	users, err := bus.store.Query(ctx, filter, p, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query user", "err", err)
		return nil, ErrUnexpected
	}
	return users, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete user", "err", err)
		return ErrUnexpected
	}
	return nil
}
