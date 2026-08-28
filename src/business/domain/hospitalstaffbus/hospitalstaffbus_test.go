package hospitalstaffbus_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus/stores/hospitalstaffdb"
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

func newTestBusiness(store hospitalstaffbus.Store) hospitalstaffbus.Business {
	log := logger.New(io.Discard, logger.LevelError, "test")
	return hospitalstaffbus.NewBusiness(log, store)
}

func newHospitalStaff() hospitalstaffbus.NewHospitalStaff {
	return hospitalstaffbus.NewHospitalStaff{
		HospitalID: 1,
		StaffID:    2,
		Role:       hospitalstaffbus.RoleDoctor,
		IsPrimary:  true,
	}
}

func TestValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{hospitalstaffbus.RoleDoctor, true},
		{hospitalstaffbus.RoleNurse, true},
		{hospitalstaffbus.RoleRegistrar, true},
		{hospitalstaffbus.RoleAdmin, true},
		{"", false},
		{"surgeon", false},
		{"Doctor", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			assert.Equal(t, tt.want, hospitalstaffbus.ValidRole(tt.role))
		})
	}
}

func TestBusiness_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		nhs := newHospitalStaff()

		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.ID == 0 &&
				hs.HospitalID == nhs.HospitalID &&
				hs.StaffID == nhs.StaffID &&
				hs.Role == nhs.Role &&
				hs.IsPrimary &&
				!hs.CreatedAt.IsZero() &&
				hs.CreatedAt.Equal(hs.UpdatedAt)
		})).Return(42, nil).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), nhs)

		require.NoError(t, err)
		assert.Equal(t, 42, hs.ID)
		assert.Equal(t, nhs.HospitalID, hs.HospitalID)
		assert.Equal(t, nhs.StaffID, hs.StaffID)
		assert.Equal(t, nhs.Role, hs.Role)
		assert.True(t, hs.IsPrimary)
		assert.False(t, hs.CreatedAt.IsZero())
		assert.Equal(t, hs.CreatedAt, hs.UpdatedAt)
		store.AssertExpectations(t)
	})

	t.Run("zero effective from defaults to now", func(t *testing.T) {
		nhs := newHospitalStaff()
		require.True(t, nhs.EffectiveFrom.IsZero())

		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(42, nil).Once()

		before := time.Now()
		hs, err := newTestBusiness(store).Create(context.Background(), nhs)
		after := time.Now()

		require.NoError(t, err)
		assert.False(t, hs.EffectiveFrom.IsZero())
		assert.False(t, hs.EffectiveFrom.Before(before))
		assert.False(t, hs.EffectiveFrom.After(after))
		store.AssertExpectations(t)
	})

	t.Run("explicit effective from is preserved", func(t *testing.T) {
		want := time.Date(2024, 3, 1, 8, 0, 0, 0, time.UTC)

		nhs := newHospitalStaff()
		nhs.EffectiveFrom = want

		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.EffectiveFrom.Equal(want)
		})).Return(42, nil).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), nhs)

		require.NoError(t, err)
		assert.True(t, hs.EffectiveFrom.Equal(want))
		store.AssertExpectations(t)
	})

	t.Run("effective to is left unset", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(42, nil).Once()

		// A zero EffectiveTo is what marks the assignment as still active.
		hs, err := newTestBusiness(store).Create(context.Background(), newHospitalStaff())

		require.NoError(t, err)
		assert.True(t, hs.EffectiveTo.IsZero())
		store.AssertExpectations(t)
	})

	t.Run("empty role is allowed", func(t *testing.T) {
		nhs := newHospitalStaff()
		nhs.Role = ""

		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.Role == ""
		})).Return(42, nil).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), nhs)

		require.NoError(t, err)
		assert.Empty(t, hs.Role)
		store.AssertExpectations(t)
	})

	t.Run("every valid role is accepted", func(t *testing.T) {
		roles := []string{
			hospitalstaffbus.RoleDoctor,
			hospitalstaffbus.RoleNurse,
			hospitalstaffbus.RoleRegistrar,
			hospitalstaffbus.RoleAdmin,
		}

		for _, role := range roles {
			t.Run(role, func(t *testing.T) {
				nhs := newHospitalStaff()
				nhs.Role = role

				store := hospitalstaffdb.NewStoreMock()
				store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
					return hs.Role == role
				})).Return(42, nil).Once()

				hs, err := newTestBusiness(store).Create(context.Background(), nhs)

				require.NoError(t, err)
				assert.Equal(t, role, hs.Role)
				store.AssertExpectations(t)
			})
		}
	})

	t.Run("unknown role maps to ErrInvalidRole without touching the store", func(t *testing.T) {
		nhs := newHospitalStaff()
		nhs.Role = "surgeon"

		store := hospitalstaffdb.NewStoreMock()

		hs, err := newTestBusiness(store).Create(context.Background(), nhs)

		require.ErrorIs(t, err, hospitalstaffbus.ErrInvalidRole)
		assert.Contains(t, err.Error(), `"surgeon"`)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("unique violation maps to ErrDuplicate", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), newHospitalStaff())

		require.ErrorIs(t, err, hospitalstaffbus.ErrDuplicate)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})

	t.Run("foreign key violation maps to ErrInvalidReference", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), newHospitalStaff())

		require.ErrorIs(t, err, hospitalstaffbus.ErrInvalidReference)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})

	t.Run("store error is passed through", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, errStore).Once()

		hs, err := newTestBusiness(store).Create(context.Background(), newHospitalStaff())

		require.ErrorIs(t, err, errStore)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})
}

func TestBusiness_RegisterStaff(t *testing.T) {
	t.Run("assigns the registrar role as primary", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.HospitalID == 1 &&
				hs.StaffID == 2 &&
				hs.Role == hospitalstaffbus.RoleRegistrar &&
				hs.IsPrimary &&
				!hs.EffectiveFrom.IsZero() &&
				hs.CreatedAt.Equal(hs.UpdatedAt)
		})).Return(42, nil).Once()

		hs, err := newTestBusiness(store).RegisterStaff(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Equal(t, 42, hs.ID)
		assert.Equal(t, 1, hs.HospitalID)
		assert.Equal(t, 2, hs.StaffID)
		assert.Equal(t, hospitalstaffbus.RoleRegistrar, hs.Role)
		assert.True(t, hs.IsPrimary)
		assert.True(t, hs.EffectiveTo.IsZero())
		assert.Equal(t, hs.CreatedAt, hs.UpdatedAt)
		store.AssertExpectations(t)
	})

	t.Run("effective from is stamped at call time", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(42, nil).Once()

		before := time.Now()
		hs, err := newTestBusiness(store).RegisterStaff(context.Background(), 1, 2)
		after := time.Now()

		require.NoError(t, err)
		assert.False(t, hs.EffectiveFrom.Before(before))
		assert.False(t, hs.EffectiveFrom.After(after))
		store.AssertExpectations(t)
	})

	t.Run("already registered maps to ErrDuplicate", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		hs, err := newTestBusiness(store).RegisterStaff(context.Background(), 1, 2)

		require.ErrorIs(t, err, hospitalstaffbus.ErrDuplicate)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})

	t.Run("unknown hospital or staff maps to ErrInvalidReference", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		hs, err := newTestBusiness(store).RegisterStaff(context.Background(), 1, 2)

		require.ErrorIs(t, err, hospitalstaffbus.ErrInvalidReference)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})

	t.Run("store error is passed through", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, errStore).Once()

		hs, err := newTestBusiness(store).RegisterStaff(context.Background(), 1, 2)

		require.ErrorIs(t, err, errStore)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, hs)
		store.AssertExpectations(t)
	})
}

func TestBusiness_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := hospitalstaffbus.HospitalStaff{
			ID:         7,
			HospitalID: 1,
			StaffID:    2,
			Role:       hospitalstaffbus.RoleNurse,
		}

		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(want, nil).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("no rows maps to ErrNotFound", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalstaffbus.HospitalStaff{}, sqldb.ErrDBNotFound).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, hospitalstaffbus.ErrNotFound)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, got)
		store.AssertExpectations(t)
	})

	t.Run("other error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalstaffbus.HospitalStaff{}, errStore).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, hospitalstaffbus.ErrUnexpected)
		assert.Equal(t, hospitalstaffbus.HospitalStaff{}, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Query(t *testing.T) {
	hospitalID := 1
	active := true
	filter := hospitalstaffbus.QueryFilter{HospitalID: &hospitalID, Active: &active}
	pg := page.MustParse(1, 10)
	orderBy := order.NewBy(hospitalstaffbus.OrderByEffectiveFrom, order.DESC)

	t.Run("success passes the filter through untouched", func(t *testing.T) {
		want := []hospitalstaffbus.HospitalStaff{{ID: 1}, {ID: 2}}

		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).Return(want, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("default order by is honoured", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, hospitalstaffbus.QueryFilter{}, pg, hospitalstaffbus.DefaultOrderBy).
			Return([]hospitalstaffbus.HospitalStaff{}, nil).Once()

		_, err := newTestBusiness(store).Query(context.Background(),
			hospitalstaffbus.QueryFilter{}, pg, hospitalstaffbus.DefaultOrderBy)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return([]hospitalstaffbus.HospitalStaff{}, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Empty(t, got)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return(nil, errStore).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.ErrorIs(t, err, hospitalstaffbus.ErrUnexpected)
		assert.Nil(t, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, hospitalstaffbus.ErrUnexpected)
		store.AssertExpectations(t)
	})

	t.Run("missing row is not reported as not found", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(sqldb.ErrDBNotFound).Once()

		// Delete has no ErrNotFound branch; every store failure collapses to
		// ErrUnexpected.
		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, hospitalstaffbus.ErrUnexpected)
		assert.NotErrorIs(t, err, hospitalstaffbus.ErrNotFound)
		store.AssertExpectations(t)
	})
}

func TestBusiness_NewWithTx(t *testing.T) {
	t.Run("success returns business backed by the tx store", func(t *testing.T) {
		tx := &txMock{}
		txStore := hospitalstaffdb.NewStoreMock()

		store := hospitalstaffdb.NewStoreMock()
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

		store := hospitalstaffdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(nil, errStore).Once()

		busTx, err := newTestBusiness(store).NewWithTx(tx)

		require.ErrorIs(t, err, errStore)
		assert.Nil(t, busTx)
		store.AssertExpectations(t)
	})
}
