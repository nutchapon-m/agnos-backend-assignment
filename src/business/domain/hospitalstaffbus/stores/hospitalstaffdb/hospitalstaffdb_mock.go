package hospitalstaffdb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
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

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (hospitalstaffbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(hospitalstaffbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, hs hospitalstaffbus.HospitalStaff) (int, error) {
	args := m.Called(ctx, hs)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (hospitalstaffbus.HospitalStaff, error) {
	args := m.Called(ctx, id)

	hs, _ := args.Get(0).(hospitalstaffbus.HospitalStaff)
	return hs, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter hospitalstaffbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalstaffbus.HospitalStaff, error) {
	args := m.Called(ctx, filter, pg, orderBy)

	hospitalStaffs, _ := args.Get(0).([]hospitalstaffbus.HospitalStaff)
	return hospitalStaffs, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, hs hospitalstaffbus.HospitalStaff) error {
	args := m.Called(ctx, hs)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
