package userdb

import (
	"context"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
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

func (m *StoreMock) NewWithTx(tx sqldb.CommitRollbacker) (userbus.Store, error) {
	args := m.Called(tx)

	store, _ := args.Get(0).(userbus.Store)
	return store, args.Error(1)
}

func (m *StoreMock) Create(ctx context.Context, u userbus.User) (int, error) {
	args := m.Called(ctx, u)
	return args.Int(0), args.Error(1)
}

func (m *StoreMock) GetByID(ctx context.Context, id int) (userbus.User, error) {
	args := m.Called(ctx, id)

	usr, _ := args.Get(0).(userbus.User)
	return usr, args.Error(1)
}

func (m *StoreMock) Query(ctx context.Context, filter userbus.QueryFilter, p page.Page, orderBy order.By) ([]userbus.User, error) {
	args := m.Called(ctx, filter, p, orderBy)

	users, _ := args.Get(0).([]userbus.User)
	return users, args.Error(1)
}

func (m *StoreMock) Update(ctx context.Context, u userbus.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *StoreMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
