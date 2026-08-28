package staffbus

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
	ErrNotFound   = errors.New("error staff not found")
	ErrDuplicate  = errors.New("error staff already exist")
	ErrUnexpected = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, s Staff) (int, error)
	GetByID(ctx context.Context, id int) (Staff, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]Staff, error)
	Update(ctx context.Context, s Staff) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, ns NewStaff) (Staff, error)
	GetByID(ctx context.Context, id int) (Staff, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]Staff, error)
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

func (bus *business) Create(ctx context.Context, ns NewStaff) (Staff, error) {
	now := time.Now()
	staff := Staff{
		UserID:       ns.UserID,
		EmployeeCode: ns.EmployeeCode,
		FirstName:    ns.FirstName,
		LastName:     ns.LastName,
		Email:        ns.Email,
		LicenseNo:    ns.LicenseNo,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := bus.store.Create(ctx, staff)
	if err != nil {
		bus.log.Error(ctx, "create staff error", "err", err)
		if errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			return Staff{}, ErrDuplicate
		}
		return Staff{}, err
	}
	staff.ID = id
	return staff, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (Staff, error) {
	staff, err := bus.store.GetByID(ctx, id)
	if err != nil {
		bus.log.Error(ctx, "error get staff", "err", err)
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return Staff{}, ErrNotFound
		}
		return Staff{}, ErrUnexpected
	}
	return staff, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]Staff, error) {
	staffs, err := bus.store.Query(ctx, filter, p, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query staff", "err", err)
		return nil, ErrUnexpected
	}
	return staffs, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete staff", "err", err)
		return ErrUnexpected
	}
	return nil
}
