package patientdb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
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

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (patientbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(patientbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, p patientbus.Patient) (int, error) {
	args := m.Called(ctx, p)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (patientbus.Patient, error) {
	args := m.Called(ctx, id)

	pt, _ := args.Get(0).(patientbus.Patient)
	return pt, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter patientbus.QueryFilter, pg page.Page, orderBy order.By) ([]patientbus.Patient, error) {
	args := m.Called(ctx, filter, pg, orderBy)

	patients, _ := args.Get(0).([]patientbus.Patient)
	return patients, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, p patientbus.Patient) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
