package staffapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/staffapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
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

const validBody = `{"user_id":1,"employee_code":"EMP-001","first_name":"Somchai","last_name":"Jaidee","email":"somchai@hospital.co.th","license_no":"MD-12345"}`

// envelope mirrors response.Response with a lazily decoded data field.
type envelope struct {
	Success bool                `json:"success"`
	Data    json.RawMessage     `json:"data"`
	Error   *response.ErrorInfo `json:"error"`
	Meta    *response.Meta      `json:"meta"`
}

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

// beginnerMock implements sqldb.Beginner and always hands out the same tx.
type beginnerMock struct {
	tx sqldb.CommitRollbacker
}

func (b *beginnerMock) Begin() (sqldb.CommitRollbacker, error) {
	return b.tx, nil
}

// newTestRouter mounts the handlers without the transaction middleware.
func newTestRouter(store staffbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store staffbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := staffapp.NewApp(staffbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/staff", append(trans, api.Create)...)
	g.GET("/staff", api.Query)
	g.GET("/staff/:id", api.GetByID)
	g.DELETE("/staff/:id", append(trans, api.Delete)...)

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
	t.Run("success", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(s staffbus.Staff) bool {
			return s.UserID == 1 &&
				s.EmployeeCode == "EMP-001" &&
				s.FirstName == "Somchai" &&
				s.LastName == "Jaidee" &&
				s.Email == "somchai@hospital.co.th" &&
				s.LicenseNo == "MD-12345" &&
				s.IsActive
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)

		var got staffapp.Staff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "EMP-001", got.EmployeeCode)
		assert.True(t, got.IsActive)
		assert.NotEmpty(t, got.CreatedAt)
		store.AssertExpectations(t)
	})

	t.Run("optional fields may be omitted", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(s staffbus.Staff) bool {
			return s.Email == "" && s.LicenseNo == ""
		})).Return(42, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff",
			`{"user_id":1,"employee_code":"EMP-002","first_name":"Somsri","last_name":"Dee"}`)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("malformed body", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff", `{"user_id":`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Argument", env.Error.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("missing user_id", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff",
			`{"employee_code":"EMP-001","first_name":"Somchai","last_name":"Jaidee"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("missing employee_code", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff",
			`{"user_id":1,"first_name":"Somchai","last_name":"Jaidee"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("invalid email", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff",
			`{"user_id":1,"employee_code":"EMP-001","first_name":"Somchai","last_name":"Jaidee","email":"not-an-email"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("duplicate maps to 409", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Staff Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(0, errStore).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/staff", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction when the middleware is in play", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := staffdb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(42, nil).Once()

		store := staffdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		r := newTestRouterWithTrans(store, &beginnerMock{tx: tx})
		w, _ := do(t, r, http.MethodPost, "/api/v1/staff", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
		txStore.AssertExpectations(t)
		tx.AssertExpectations(t)
		// The write must go through the tx-scoped store only.
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{ID: 7, UserID: 1, EmployeeCode: "EMP-001"}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got staffapp.Staff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 7, got.ID)
		assert.Equal(t, "EMP-001", got.EmployeeCode)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Staff Not Found", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("unexpected error maps to 500", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("non numeric id", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("zero id", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff/0", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		staffs := []staffbus.Staff{{ID: 1, EmployeeCode: "EMP-001"}, {ID: 2, EmployeeCode: "EMP-002"}}

		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, staffbus.QueryFilter{}, page.MustParse(1, 10), staffbus.DefaultOrderBy).
			Return(staffs, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []staffapp.Staff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, "EMP-001", got[0].EmployeeCode)

		require.NotNil(t, env.Meta)
		assert.Equal(t, 1, env.Meta.Page)
		assert.Equal(t, 10, env.Meta.PerPage)
		assert.Equal(t, 2, env.Meta.Total)
		store.AssertExpectations(t)
	})

	t.Run("honours filter, paging and ordering", func(t *testing.T) {
		id, userID, pgNum, limit := 5, 3, 2, 20
		orderBy := "created_at,DESC"

		want := staffbus.QueryFilter{ID: &id, UserID: &userID, OrderBy: &orderBy, Page: &pgNum, Limit: &limit}

		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(2, 20), order.NewBy(staffbus.OrderByCreatedAt, order.DESC)).
			Return([]staffbus.Staff{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/staff?id=5&user_id=3&page=2&limit=20&order_by=created_at,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("limit above the maximum", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff?limit=1000", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("non numeric query param", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff?user_id=abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/staff", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown staff is not deleted", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("bad id", func(t *testing.T) {
		store := staffdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/staff/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := staffdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}
