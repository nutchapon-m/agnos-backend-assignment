package hospitalpatientapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/hospitalpatientapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
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

const validBody = `{"hospital_id":1,"patient_id":2,"hn":"HN-0001"}`

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

func newTestRouter(store hospitalpatientbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store hospitalpatientbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := hospitalpatientapp.NewApp(hospitalpatientbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/hospital-patient", append(trans, api.Create)...)
	g.GET("/hospital-patient", api.Query)
	g.GET("/hospital-patient/:id", api.GetByID)
	g.DELETE("/hospital-patient/:id", append(trans, api.Delete)...)

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
	t.Run("success defaults status and registered_at", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.HospitalID == 1 &&
				hp.PatientID == 2 &&
				hp.HN == "HN-0001" &&
				hp.Status == hospitalpatientbus.StatusActive &&
				!hp.RegisteredAt.IsZero()
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient", validBody)

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalpatientapp.HospitalPatient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "active", got.Status)
		assert.NotEmpty(t, got.RegisteredAt)
		store.AssertExpectations(t)
	})

	t.Run("explicit status is kept", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(hp hospitalpatientbus.HospitalPatient) bool {
			return hp.Status == hospitalpatientbus.StatusInactive
		})).Return(42, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient",
			`{"hospital_id":1,"patient_id":2,"hn":"HN-0001","status":"inactive"}`)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("missing hospital_id", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient",
			`{"patient_id":2,"hn":"HN-0001"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("unknown status", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient",
			`{"hospital_id":1,"patient_id":2,"hn":"HN-0001","status":"pending"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("bad registered_at format", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient",
			`{"hospital_id":1,"patient_id":2,"hn":"HN-0001","registered_at":"2024-01-02"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("duplicate registration maps to 409", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Patient Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown hospital or patient maps to 422", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient", validBody)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Reference", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(0, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital-patient", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := hospitalpatientdb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("hospitalpatientbus.HospitalPatient")).
			Return(42, nil).Once()

		store := hospitalpatientdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		w, _ := do(t, newTestRouterWithTrans(store, &beginnerMock{tx: tx}),
			http.MethodPost, "/api/v1/hospital-patient", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		tx.AssertExpectations(t)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalpatientbus.HospitalPatient{ID: 7, HospitalID: 1, PatientID: 2, HN: "HN-0001"}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalpatientapp.HospitalPatient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, "HN-0001", got.HN)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalpatientbus.HospitalPatient{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		store.AssertExpectations(t)
	})

	t.Run("bad id", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		rows := []hospitalpatientbus.HospitalPatient{{ID: 1}, {ID: 2}}

		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, hospitalpatientbus.QueryFilter{}, page.MustParse(1, 10), hospitalpatientbus.DefaultOrderBy).
			Return(rows, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []hospitalpatientapp.HospitalPatient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, 2, env.Meta.Total)
		store.AssertExpectations(t)
	})

	t.Run("lists the patients of one hospital", func(t *testing.T) {
		hospitalID := 1
		status := "active"
		orderBy := "registered_at,DESC"

		want := hospitalpatientbus.QueryFilter{HospitalID: &hospitalID, Status: &status, OrderBy: &orderBy}

		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(1, 10), order.NewBy(hospitalpatientbus.OrderByRegisteredAt, order.DESC)).
			Return([]hospitalpatientbus.HospitalPatient{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/hospital-patient?hospital_id=1&status=active&order_by=registered_at,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital-patient", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(hospitalpatientbus.HospitalPatient{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital-patient/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown registration is not deleted", func(t *testing.T) {
		store := hospitalpatientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalpatientbus.HospitalPatient{}, sqldb.ErrDBNotFound).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital-patient/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}
