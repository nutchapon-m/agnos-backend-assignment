package hospitalstaffapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/hospitalstaffapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
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

const validBody = `{"hospital_id":1,"staff_id":2,"role":"doctor","is_primary":true,"effective_from":"2024-01-02"}`

type envelope struct {
	Success bool                `json:"success"`
	Data    json.RawMessage     `json:"data"`
	Error   *response.ErrorInfo `json:"error"`
	Meta    *response.Meta      `json:"meta"`
}

type txMock struct {
	mock.Mock
}

func (m *txMock) Commit() error {
	return m.Called().Error(0)
}

func (m *txMock) Rollback() error {
	return m.Called().Error(0)
}

type beginnerMock struct {
	tx sqldb.CommitRollbacker
}

func (b *beginnerMock) Begin() (sqldb.CommitRollbacker, error) {
	return b.tx, nil
}

func newTestRouter(store hospitalstaffbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store hospitalstaffbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := hospitalstaffapp.NewApp(hospitalstaffbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/hospital-staff", append(trans, api.Create)...)
	g.GET("/hospital-staff", api.Query)
	g.GET("/hospital-staff/:id", api.GetByID)
	g.DELETE("/hospital-staff/:id", append(trans, api.Delete)...)

	return r
}

func do(t *testing.T, r *gin.Engine, method, target, body string) (*httptest.ResponseRecorder, envelope) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body: %s", w.Body.String())

	return w, env
}

func TestApp_Create(t *testing.T) {
	t.Run("success round trips the effective date", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.HospitalID == 1 &&
				hs.StaffID == 2 &&
				hs.Role == hospitalstaffbus.RoleDoctor &&
				hs.IsPrimary &&
				hs.EffectiveFrom.Equal(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)) &&
				hs.EffectiveTo.IsZero()
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff", validBody)

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalstaffapp.HospitalStaff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "2024-01-02", got.EffectiveFrom)
		assert.Empty(t, got.EffectiveTo)
		store.AssertExpectations(t)
	})

	t.Run("effective_from defaults to today", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return !hs.EffectiveFrom.IsZero()
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff",
			`{"hospital_id":1,"staff_id":2,"role":"nurse"}`)

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalstaffapp.HospitalStaff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, time.Now().Format(time.DateOnly), got.EffectiveFrom)
		store.AssertExpectations(t)
	})

	t.Run("unknown role", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff",
			`{"hospital_id":1,"staff_id":2,"role":"janitor"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("missing staff_id", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff",
			`{"hospital_id":1,"role":"doctor"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("bad effective_from format", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff",
			`{"hospital_id":1,"staff_id":2,"effective_from":"02/01/2024"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("already assigned maps to 409", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Staff Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown hospital or staff maps to 422", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff", validBody)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Reference", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-staff", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := hospitalstaffdb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(42, nil).Once()

		store := hospitalstaffdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		w, _ := do(t, newTestRouterWithTrans(store, &beginnerMock{tx: tx}),
			http.MethodPost, "/api/v1/hospital-staff", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		tx.AssertExpectations(t)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success exposes a closed assignment", func(t *testing.T) {
		effectiveTo := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalstaffbus.HospitalStaff{ID: 7, HospitalID: 1, StaffID: 2, EffectiveTo: effectiveTo}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalstaffapp.HospitalStaff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, "2024-06-30", got.EffectiveTo)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalstaffbus.HospitalStaff{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		store.AssertExpectations(t)
	})

	t.Run("bad id", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		rows := []hospitalstaffbus.HospitalStaff{{ID: 1}, {ID: 2}}

		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, hospitalstaffbus.QueryFilter{}, page.MustParse(1, 10), hospitalstaffbus.DefaultOrderBy).
			Return(rows, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []hospitalstaffapp.HospitalStaff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, 2, env.Meta.Total)
		store.AssertExpectations(t)
	})

	t.Run("lists the active assignments of one staff", func(t *testing.T) {
		staffID := 2
		active := true
		role := "doctor"
		orderBy := "effective_from,DESC"

		want := hospitalstaffbus.QueryFilter{StaffID: &staffID, Role: &role, Active: &active, OrderBy: &orderBy}

		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(1, 10), order.NewBy(hospitalstaffbus.OrderByEffectiveFrom, order.DESC)).
			Return([]hospitalstaffbus.HospitalStaff{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/hospital-staff?staff_id=2&role=doctor&active=true&order_by=effective_from,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-staff", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(hospitalstaffbus.HospitalStaff{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital-staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown assignment is not deleted", func(t *testing.T) {
		store := hospitalstaffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalstaffbus.HospitalStaff{}, sqldb.ErrDBNotFound).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital-staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}

func TestValidRole(t *testing.T) {
	for _, role := range []string{
		hospitalstaffbus.RoleDoctor,
		hospitalstaffbus.RoleNurse,
		hospitalstaffbus.RoleRegistrar,
		hospitalstaffbus.RoleAdmin,
	} {
		assert.True(t, hospitalstaffbus.ValidRole(role), role)
	}

	assert.False(t, hospitalstaffbus.ValidRole("janitor"))
	assert.False(t, hospitalstaffbus.ValidRole(""))
}
