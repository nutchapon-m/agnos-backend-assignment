package userapp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/userapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus/stores/userdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errStore = errors.New("store failure")

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

func newTestLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelError, "test")
}

// newTestRouter mounts the handlers without the transaction middleware.
func newTestRouter(store userbus.Store) *gin.Engine {
	return newTestRouterWithTrans(store, nil)
}

func newTestRouterWithTrans(store userbus.Store, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := newTestLogger()
	api := userapp.NewApp(userbus.NewBusiness(log, store))

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")
	g.POST("/user", append(trans, api.Create)...)
	g.GET("/user", api.Query)
	g.GET("/user/:id", api.GetByID)
	g.DELETE("/user/:id", append(trans, api.Delete)...)

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
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(u userbus.User) bool {
			return u.Username == "gopher" && u.Password == "secret1"
		})).Return(42, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user",
			`{"username":"gopher","password":"secret1"}`)

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)

		var got userapp.User
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.ID)
		assert.Equal(t, "gopher", got.Username)
		assert.NotEmpty(t, got.CreatedAt)

		// The password must never leak back to the caller.
		assert.NotContains(t, w.Body.String(), "secret1")
		store.AssertExpectations(t)
	})

	t.Run("malformed body", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user", `{"username":`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Argument", env.Error.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("validation rejects short password before hitting the business", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user",
			`{"username":"gopher","password":"123"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("missing username", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user",
			`{"password":"secret1"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("duplicate maps to 409", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user",
			`{"username":"gopher","password":"secret1"}`)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "User Already Exist", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(0, errStore).Once()

		w, env := do(t, newTestRouter(store), http.MethodPost, "/api/v1/user",
			`{"username":"gopher","password":"secret1"}`)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("runs on the request transaction when the middleware is in play", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txStore := userdb.NewStoreMock()
		txStore.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(42, nil).Once()

		store := userdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(txStore, nil).Once()

		r := newTestRouterWithTrans(store, &beginnerMock{tx: tx})
		w, _ := do(t, r, http.MethodPost, "/api/v1/user", `{"username":"gopher","password":"secret1"}`)

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
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(userbus.User{ID: 7, Username: "gopher"}, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got userapp.User
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 7, got.ID)
		assert.Equal(t, "gopher", got.Username)
		store.AssertExpectations(t)
	})

	t.Run("not found maps to 404 and does not fall through", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(userbus.User{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "User Not Found", env.Error.Code)
		store.AssertExpectations(t)
	})

	t.Run("unexpected error maps to 500", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(userbus.User{}, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("non numeric id", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("zero id", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user/0", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		users := []userbus.User{{ID: 1, Username: "a"}, {ID: 2, Username: "b"}}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, userbus.QueryFilter{}, page.MustParse(1, 10), userbus.DefaultOrderBy).
			Return(users, nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []userapp.User
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, "a", got[0].Username)

		require.NotNil(t, env.Meta)
		assert.Equal(t, 1, env.Meta.Page)
		assert.Equal(t, 10, env.Meta.PerPage)
		assert.Equal(t, 2, env.Meta.Total)
		store.AssertExpectations(t)
	})

	t.Run("honours filter, paging and ordering", func(t *testing.T) {
		id, pgNum, limit := 5, 2, 20
		orderBy := "created_at,DESC"

		want := userbus.QueryFilter{ID: &id, OrderBy: &orderBy, Page: &pgNum, Limit: &limit}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, want, page.MustParse(2, 20), order.NewBy(userbus.OrderByCreatedAt, order.DESC)).
			Return([]userbus.User{}, nil).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet,
			"/api/v1/user?id=5&page=2&limit=20&order_by=created_at,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("limit above the maximum", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user?limit=1000", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("non numeric query param", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user?page=abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodGet, "/api/v1/user", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(userbus.User{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/user/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		store.AssertExpectations(t)
	})

	t.Run("unknown user is not deleted", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(userbus.User{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/user/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		store.AssertExpectations(t)
		store.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("bad id", func(t *testing.T) {
		store := userdb.NewStoreMock()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/user/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		store.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(userbus.User{ID: 7}, nil).Once()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		w, _ := do(t, newTestRouter(store), http.MethodDelete, "/api/v1/user/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		store.AssertExpectations(t)
	})
}
