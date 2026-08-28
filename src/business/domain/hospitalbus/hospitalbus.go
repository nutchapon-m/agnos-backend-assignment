package hospitalbus

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
	ErrNotFound   = errors.New("error hospital not found")
	ErrDuplicate  = errors.New("error hospital already exist")
	ErrUnexpected = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, h Hospital) (int, error)
	GetByID(ctx context.Context, id int) (Hospital, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Hospital, error)
	Update(ctx context.Context, h Hospital) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, nh NewHospital) (Hospital, error)
	GetByID(ctx context.Context, id int) (Hospital, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Hospital, error)
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

func (bus *business) Create(ctx context.Context, nh NewHospital) (Hospital, error) {
	now := time.Now()
	hospital := Hospital{
		Code:         nh.Code,
		Name:         nh.Name,
		ProvinceCode: nh.ProvinceCode,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := bus.store.Create(ctx, hospital)
	if err != nil {
		bus.log.Error(ctx, "create hospital error", "err", err)
		if errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			return Hospital{}, ErrDuplicate
		}
		return Hospital{}, err
	}
	hospital.ID = id
	return hospital, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (Hospital, error) {
	hospital, err := bus.store.GetByID(ctx, id)
	if err != nil {
		bus.log.Error(ctx, "error get hospital", "err", err)
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return Hospital{}, ErrNotFound
		}
		return Hospital{}, ErrUnexpected
	}
	return hospital, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Hospital, error) {
	hospitals, err := bus.store.Query(ctx, filter, pg, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query hospital", "err", err)
		return nil, ErrUnexpected
	}
	return hospitals, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete hospital", "err", err)
		return ErrUnexpected
	}
	return nil
}
