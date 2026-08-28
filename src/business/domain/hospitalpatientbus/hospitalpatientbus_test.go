package hospitalpatientbus_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus/stores/hospitalpatientdb"
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

func newTestBusiness(store hospitalpatientbus.Store) hospitalpatientbus.Business {
	log := logger.New(io.Discard, logger.LevelError, "test")
	return hospitalpatientbus.NewBusiness(log, store)
}

func newHospitalPatient() hospitalpatientbus.NewHospitalPatient {
	return hospitalpatientbus.NewHospitalPatient{
		HospitalID: 1,
		PatientID:  2,
		HN:         "HN-0001",
	}
}

func TestBusiness_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		nhp := newHospitalPatient()

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.ID == 0 &&
				hp.HospitalID == nhp.HospitalID &&
				hp.PatientID == nhp.PatientID &&
				hp.HN == nhp.HN &&
				!hp.CreatedAt.IsZero() &&
				hp.CreatedAt.Equal(hp.UpdatedAt)
		})).Return(42, nil).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.Equal(t, 42, hp.ID)
		assert.Equal(t, nhp.HospitalID, hp.HospitalID)
		assert.Equal(t, nhp.PatientID, hp.PatientID)
		assert.Equal(t, nhp.HN, hp.HN)
		assert.False(t, hp.CreatedAt.IsZero())
		assert.Equal(t, hp.CreatedAt, hp.UpdatedAt)
		store.AssertExpectations(t)
	})

	t.Run("empty status defaults to active", func(t *testing.T) {
		nhp := newHospitalPatient()
		require.Empty(t, nhp.Status)

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.Status == hospitalpatientbus.StatusActive
		})).Return(42, nil).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.Equal(t, hospitalpatientbus.StatusActive, hp.Status)
		store.AssertExpectations(t)
	})

	t.Run("explicit status is preserved", func(t *testing.T) {
		nhp := newHospitalPatient()
		nhp.Status = hospitalpatientbus.StatusInactive

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.Status == hospitalpatientbus.StatusInactive
		})).Return(42, nil).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.Equal(t, hospitalpatientbus.StatusInactive, hp.Status)
		store.AssertExpectations(t)
	})

	t.Run("unknown status is not validated", func(t *testing.T) {
		nhp := newHospitalPatient()
		nhp.Status = "archived"

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.Status == "archived"
		})).Return(42, nil).Once()

		// The business only fills in a default; an out-of-range status reaches
		// the store and is left to the table's check constraint.
		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.Equal(t, "archived", hp.Status)
		store.AssertExpectations(t)
	})

	t.Run("zero registered at defaults to now", func(t *testing.T) {
		nhp := newHospitalPatient()
		require.True(t, nhp.RegisteredAt.IsZero())

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(42, nil).Once()

		before := time.Now()
		hp, err := newTestBusiness(store).Create(context.Background(), nhp)
		after := time.Now()

		require.NoError(t, err)
		assert.False(t, hp.RegisteredAt.IsZero())
		assert.False(t, hp.RegisteredAt.Before(before))
		assert.False(t, hp.RegisteredAt.After(after))
		store.AssertExpectations(t)
	})

	t.Run("explicit registered at is preserved", func(t *testing.T) {
		want := time.Date(2024, 3, 1, 8, 0, 0, 0, time.UTC)

		nhp := newHospitalPatient()
		nhp.RegisteredAt = want

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.RegisteredAt.Equal(want)
		})).Return(42, nil).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.True(t, hp.RegisteredAt.Equal(want))
		// A backdated registration must not drag the audit timestamps with it.
		assert.False(t, hp.CreatedAt.Equal(want))
		store.AssertExpectations(t)
	})

	t.Run("empty hn is passed through", func(t *testing.T) {
		nhp := newHospitalPatient()
		nhp.HN = ""

		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.HN == ""
		})).Return(42, nil).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), nhp)

		require.NoError(t, err)
		assert.Empty(t, hp.HN)
		store.AssertExpectations(t)
	})

	t.Run("duplicate registration maps to ErrDuplicate", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), newHospitalPatient())

		require.ErrorIs(t, err, hospitalpatientbus.ErrDuplicate)
		assert.Equal(t, hospitalpatientbus.HospitalPatient{}, hp)
		store.AssertExpectations(t)
	})

	t.Run("unknown hospital or patient maps to ErrInvalidReference", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), newHospitalPatient())

		require.ErrorIs(t, err, hospitalpatientbus.ErrInvalidReference)
		assert.Equal(t, hospitalpatientbus.HospitalPatient{}, hp)
		store.AssertExpectations(t)
	})

	t.Run("store error is passed through", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, errStore).Once()

		hp, err := newTestBusiness(store).Create(context.Background(), newHospitalPatient())

		require.ErrorIs(t, err, errStore)
		assert.Equal(t, hospitalpatientbus.HospitalPatient{}, hp)
		store.AssertExpectations(t)
	})
}

func TestBusiness_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := hospitalpatientbus.HospitalPatient{
			ID:         7,
			HospitalID: 1,
			PatientID:  2,
			HN:         "HN-0001",
			Status:     hospitalpatientbus.StatusActive,
		}

		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(want, nil).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("no rows maps to ErrNotFound", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalpatientbus.HospitalPatient{}, sqldb.ErrDBNotFound).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, hospitalpatientbus.ErrNotFound)
		assert.Equal(t, hospitalpatientbus.HospitalPatient{}, got)
		store.AssertExpectations(t)
	})

	t.Run("other error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalpatientbus.HospitalPatient{}, errStore).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, hospitalpatientbus.ErrUnexpected)
		assert.NotErrorIs(t, err, errStore)
		assert.Equal(t, hospitalpatientbus.HospitalPatient{}, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Query(t *testing.T) {
	patientID := 2
	status := hospitalpatientbus.StatusActive
	filter := hospitalpatientbus.QueryFilter{PatientID: &patientID, Status: &status}
	pg := page.MustParse(1, 10)
	orderBy := order.NewBy(hospitalpatientbus.OrderByRegisteredAt, order.DESC)

	t.Run("success passes the filter through untouched", func(t *testing.T) {
		want := []hospitalpatientbus.HospitalPatient{
			{ID: 1, PatientID: 2, HN: "HN-0001"},
			{ID: 2, PatientID: 2, HN: "HN-0002"},
		}

		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).Return(want, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("default order by is honoured", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, hospitalpatientbus.QueryFilter{}, pg, hospitalpatientbus.DefaultOrderBy).
			Return([]hospitalpatientbus.HospitalPatient{}, nil).Once()

		_, err := newTestBusiness(store).Query(context.Background(),
			hospitalpatientbus.QueryFilter{}, pg, hospitalpatientbus.DefaultOrderBy)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("zero page is passed through", func(t *testing.T) {
		// patientapp.GetByID queries with a zero page to pull every registration.
		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, page.Page{}, hospitalpatientbus.DefaultOrderBy).
			Return([]hospitalpatientbus.HospitalPatient{{ID: 1}}, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(),
			filter, page.Page{}, hospitalpatientbus.DefaultOrderBy)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		store.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return([]hospitalpatientbus.HospitalPatient{}, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Empty(t, got)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return(nil, errStore).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.ErrorIs(t, err, hospitalpatientbus.ErrUnexpected)
		assert.Nil(t, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, hospitalpatientbus.ErrUnexpected)
		store.AssertExpectations(t)
	})

	t.Run("missing row is not reported as not found", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(sqldb.ErrDBNotFound).Once()

		// Delete has no ErrNotFound branch; every store failure collapses to
		// ErrUnexpected.
		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, hospitalpatientbus.ErrUnexpected)
		assert.NotErrorIs(t, err, hospitalpatientbus.ErrNotFound)
		store.AssertExpectations(t)
	})
}

func TestBusiness_NewWithTx(t *testing.T) {
	t.Run("success returns business backed by the tx store", func(t *testing.T) {
		tx := &txMock{}
		txStore := hospitalpatientdb.NewStoreMock()

		store := hospitalpatientdb.NewStoreMock()
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

		store := hospitalpatientdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(nil, errStore).Once()

		busTx, err := newTestBusiness(store).NewWithTx(tx)

		require.ErrorIs(t, err, errStore)
		assert.Nil(t, busTx)
		store.AssertExpectations(t)
	})
}
