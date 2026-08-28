package hospitalapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/hospitalapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus/stores/hospitaldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errStore = errors.New("store failure")

const validBody = `{"code":"H001","name":"Bangkok Hospital","province_code":"10"}`

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

func newTestRouter(store hospitalbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store hospitalbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := hospitalapp.NewApp(hospitalbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/hospital", append(trans, api.Create)...)
	g.GET("/hospital", api.Query)
	g.GET("/hospital/:id", api.GetByID)
	g.DELETE("/hospital/:id", append(trans, api.Delete)...)

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
	t.Run("success defaults to active", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(h hospitalbus.Hospital) bool {
			return h.Code == "H001" && h.Name == "Bangkok Hospital" && h.ProvinceCode == "10" && h.IsActive
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital", validBody)

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalapp.Hospital
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "H001", got.Code)
		assert.True(t, got.IsActive)
		store.AssertExpectations(t)
	})

	t.Run("missing code", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital", `{"name":"No Code"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("invalid province code", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital",
			`{"code":"H001","province_code":"bangkok"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("duplicate code maps to 409", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalbus.Hospital")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("hospitalbus.Hospital")).
			Return(0, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/hospital", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := hospitaldb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("hospitalbus.Hospital")).
			Return(42, nil).Once()

		store := hospitaldb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		w, _ := do(t, newTestRouterWithTrans(store, &beginnerMock{tx: tx}),
			http.MethodPost, "/api/v1/hospital", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		tx.AssertExpectations(t)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalbus.Hospital{ID: 7, Code: "H001", IsActive: true}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got hospitalapp.Hospital
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, "H001", got.Code)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(hospitalbus.Hospital{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Not Found", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("bad id", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		hospitals := []hospitalbus.Hospital{{ID: 1, Code: "H001"}, {ID: 2, Code: "H002"}}

		store := hospitaldb.NewStoreMock()
		store.On("Query", mock.Anything, hospitalbus.QueryFilter{}, page.MustParse(1, 10), hospitalbus.DefaultOrderBy).
			Return(hospitals, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []hospitalapp.Hospital
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, 2, env.Meta.Total)
		store.AssertExpectations(t)
	})

	t.Run("filters by province and active flag", func(t *testing.T) {
		provinceCode := "10"
		isActive := true
		orderBy := "code"

		want := hospitalbus.QueryFilter{ProvinceCode: &provinceCode, IsActive: &isActive, OrderBy: &orderBy}

		store := hospitaldb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(1, 10), order.NewBy(hospitalbus.OrderByCode, order.ASC)).
			Return([]hospitalbus.Hospital{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/hospital?province_code=10&is_active=true&order_by=code", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/hospital", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(hospitalbus.Hospital{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown hospital is not deleted", func(t *testing.T) {
		store := hospitaldb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(hospitalbus.Hospital{}, sqldb.ErrDBNotFound).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/hospital/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}
