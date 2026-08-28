package hospitalpatientdb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
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

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (hospitalpatientbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(hospitalpatientbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, hp hospitalpatientbus.HospitalPatient) (int, error) {
	args := m.Called(ctx, hp)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (hospitalpatientbus.HospitalPatient, error) {
	args := m.Called(ctx, id)

	hp, _ := args.Get(0).(hospitalpatientbus.HospitalPatient)
	return hp, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter hospitalpatientbus.QueryFilter, pg page.Page, orderBy order.By) ([]hospitalpatientbus.HospitalPatient, error) {
	args := m.Called(ctx, filter, pg, orderBy)

	hospitalPatients, _ := args.Get(0).([]hospitalpatientbus.HospitalPatient)
	return hospitalPatients, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, hp hospitalpatientbus.HospitalPatient) error {
	args := m.Called(ctx, hp)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
