package staffdb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
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

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (staffbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(staffbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, s staffbus.Staff) (int, error) {
	args := m.Called(ctx, s)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (staffbus.Staff, error) {
	args := m.Called(ctx, id)

	stf, _ := args.Get(0).(staffbus.Staff)
	return stf, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter staffbus.QueryFilter, p page.Page, orderBy order.By) ([]staffbus.Staff, error) {
	args := m.Called(ctx, filter, p, orderBy)

	staffs, _ := args.Get(0).([]staffbus.Staff)
	return staffs, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, s staffbus.Staff) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
