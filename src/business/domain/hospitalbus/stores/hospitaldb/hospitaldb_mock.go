package hospitaldb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/stretchr/testify/mock"
)

type StoreMock struct {
	mock.Mock
}

func NewStoreMock() *StoreMock {
	return &StoreMock{}
}

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (hospitalbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(hospitalbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, h hospitalbus.Hospital) (int, error) {
	args := m.Called(ctx, h)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (hospitalbus.Hospital, error) {
	args := m.Called(ctx, id)

	hsp, _ := args.Get(0).(hospitalbus.Hospital)
	return hsp, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter hospitalbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalbus.Hospital, error) {
	args := m.Called(ctx, filter, pg, orderBy)

	hospitals, _ := args.Get(0).([]hospitalbus.Hospital)
	return hospitals, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, h hospitalbus.Hospital) error {
	args := m.Called(ctx, h)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
