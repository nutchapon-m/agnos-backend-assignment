package staffbus_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus/stores/staffdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errStore = errors.New("store failure")

// txMock implements sqldb.CommitRollbacker.
type txMock struct {
	mock.Mock
}

func (m *txMock) Commit() error {
	return m.Called().Error(0)
}

func (m *txMock) Rollback() error {
	return m.Called().Error(0)
}

func newTestBusiness(store staffbus.Store) staffbus.Business {
	log := logger.New(io.Discard, logger.LevelError, "test")
	return staffbus.NewBusiness(log, store)
}

func newStaff() staffbus.NewStaff {
	return staffbus.NewStaff{
		UserID:       1,
		EmployeeCode: "EMP-001",
		FirstName:    "Somchai",
		LastName:     "Jaidee",
		Email:        "somchai@hospital.co.th",
		LicenseNo:    "MD-12345",
	}
}

func TestBusiness_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ns := newStaff()

		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(s staffbus.Staff) bool {
			return s.UserID == ns.UserID &&
				s.EmployeeCode == ns.EmployeeCode &&
				s.FirstName == ns.FirstName &&
				s.LastName == ns.LastName &&
				s.Email == ns.Email &&
				s.LicenseNo == ns.LicenseNo &&
				s.IsActive &&
				!s.CreatedAt.IsZero() &&
				s.CreatedAt.Equal(s.UpdatedAt)
		})).Return(42, nil).Once()

		stf, err := newTestBusiness(store).Create(context.Background(), ns)

		require.NoError(t, err)
		assert.Equal(t, 42, stf.ID)
		assert.Equal(t, ns.UserID, stf.UserID)
		assert.Equal(t, ns.EmployeeCode, stf.EmployeeCode)
		assert.Equal(t, ns.Email, stf.Email)
		assert.True(t, stf.IsActive)
		assert.False(t, stf.CreatedAt.IsZero())
		assert.Equal(t, stf.CreatedAt, stf.UpdatedAt)
		store.AssertExpectations(t)
	})

	t.Run("unique violation maps to ErrDuplicate", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		stf, err := newTestBusiness(store).Create(context.Background(), newStaff())

		require.ErrorIs(t, err, staffbus.ErrDuplicate)
		assert.Equal(t, staffbus.Staff{}, stf)
		store.AssertExpectations(t)
	})

	t.Run("store error returns zero staff and passes error through", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(0, errStore).Once()

		stf, err := newTestBusiness(store).Create(context.Background(), newStaff())

		require.ErrorIs(t, err, errStore)
		assert.Equal(t, staffbus.Staff{}, stf)
		store.AssertExpectations(t)
	})
}

func TestBusiness_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := staffbus.Staff{ID: 7, UserID: 1, EmployeeCode: "EMP-001"}

		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(want, nil).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("no rows maps to ErrNotFound", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, sqldb.ErrDBNotFound).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, staffbus.ErrNotFound)
		assert.Equal(t, staffbus.Staff{}, got)
		store.AssertExpectations(t)
	})

	t.Run("other error maps to ErrUnexpected", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, errStore).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, staffbus.ErrUnexpected)
		assert.Equal(t, staffbus.Staff{}, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Query(t *testing.T) {
	userID := 1
	filter := staffbus.QueryFilter{UserID: &userID}
	pg := page.MustParse(1, 10)
	orderBy := order.NewBy(staffbus.OrderByID, order.ASC)

	t.Run("success", func(t *testing.T) {
		want := []staffbus.Staff{{ID: 1, EmployeeCode: "EMP-001"}, {ID: 2, EmployeeCode: "EMP-002"}}

		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).Return(want, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return([]staffbus.Staff{}, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Empty(t, got)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return(nil, errStore).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.ErrorIs(t, err, staffbus.ErrUnexpected)
		assert.Nil(t, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, staffbus.ErrUnexpected)
		store.AssertExpectations(t)
	})
}

func TestBusiness_NewWithTx(t *testing.T) {
	t.Run("success returns business backed by the tx store", func(t *testing.T) {
		tx := &txMock{}
		txStore := staffdb.NewStoreMock()

		store := staffdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		// The returned business must delegate to the tx-scoped store, not the original.
		txStore.On("Delete", mock.Anything, 7).Return(nil).Once()

		busTx, err := newTestBusiness(store).NewWithTx(tx)

		require.NoError(t, err)
		require.NotNil(t, busTx)
		require.NoError(t, busTx.Delete(context.Background(), 7))

		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("store error is passed through", func(t *testing.T) {
		tx := &txMock{}

		store := staffdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(nil, errStore).Once()

		busTx, err := newTestBusiness(store).NewWithTx(tx)

		require.ErrorIs(t, err, errStore)
		assert.Nil(t, busTx)
		store.AssertExpectations(t)
	})
}
