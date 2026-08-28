package patientapp_test

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
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/patientapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus/stores/patientdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errStore = errors.New("store failure")

const validBody = `{"national_id":"1234567890123","first_name_th":"สมชาย","last_name_th":"ใจดี","date_of_birth":"1990-05-04","gender":"M","phone":"0812345678"}`

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

func newTestRouter(store patientbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store patientbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := patientapp.NewApp(patientbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/patient", append(trans, api.Create)...)
	g.GET("/patient", api.Query)
	g.GET("/patient/:id", api.GetByID)
	g.DELETE("/patient/:id", append(trans, api.Delete)...)

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
	t.Run("success round trips the date of birth", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(p patientbus.Patient) bool {
			return p.NationalID == "1234567890123" &&
				p.FirstNameTH == "สมชาย" &&
				p.Gender == patientbus.GenderMale &&
				p.DateOfBirth.Equal(time.Date(1990, 5, 4, 0, 0, 0, 0, time.UTC))
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient", validBody)

		require.Equal(t, http.StatusOK, w.Code)

		var got patientapp.Patient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "1990-05-04", got.DateOfBirth)
		store.AssertExpectations(t)
	})

	t.Run("date of birth may be omitted", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(p patientbus.Patient) bool {
			return p.DateOfBirth.IsZero()
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient",
			`{"first_name_th":"สมศรี","last_name_th":"ดี"}`)

		require.Equal(t, http.StatusOK, w.Code)

		var got patientapp.Patient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Empty(t, got.DateOfBirth)
		store.AssertExpectations(t)
	})

	t.Run("missing thai name", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient", `{"gender":"M"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("invalid gender", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient",
			`{"first_name_th":"a","last_name_th":"b","gender":"X"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("invalid national id length", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient",
			`{"first_name_th":"a","last_name_th":"b","national_id":"123"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("bad date format", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient",
			`{"first_name_th":"a","last_name_th":"b","date_of_birth":"04/05/1990"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("duplicate national id maps to 409", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("patientbus.Patient")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Patient Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("patientbus.Patient")).
			Return(0, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/patient", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := patientdb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("patientbus.Patient")).
			Return(42, nil).Once()

		store := patientdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		w, _ := do(t, newTestRouterWithTrans(store, &beginnerMock{tx: tx}),
			http.MethodPost, "/api/v1/patient", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		tx.AssertExpectations(t)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(patientbus.Patient{ID: 7, FirstNameTH: "สมชาย"}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got patientapp.Patient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 7, got.ID)
		assert.Equal(t, "สมชาย", got.FirstNameTH)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(patientbus.Patient{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Patient Not Found", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("bad id", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		patients := []patientbus.Patient{{ID: 1}, {ID: 2}}

		store := patientdb.NewStoreMock()
		store.On("Query", mock.Anything, patientbus.QueryFilter{}, page.MustParse(1, 10), patientbus.DefaultOrderBy).
			Return(patients, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []patientapp.Patient
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)

		require.NotNil(t, env.Meta)
		assert.Equal(t, 1, env.Meta.Page)
		assert.Equal(t, 10, env.Meta.PerPage)
		store.AssertExpectations(t)
	})

	t.Run("searches by national id", func(t *testing.T) {
		nationalID := "1234567890123"
		pgNum, limit := 2, 20
		orderBy := "created_at,DESC"

		want := patientbus.QueryFilter{NationalID: &nationalID, OrderBy: &orderBy, Page: &pgNum, Limit: &limit}

		store := patientdb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(2, 20), order.NewBy(patientbus.OrderByCreatedAt, order.DESC)).
			Return([]patientbus.Patient{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/patient?national_id=1234567890123&page=2&limit=20&order_by=created_at,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := patientdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/patient", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(patientbus.Patient{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/patient/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown patient is not deleted", func(t *testing.T) {
		store := patientdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(patientbus.Patient{}, sqldb.ErrDBNotFound).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/patient/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}
