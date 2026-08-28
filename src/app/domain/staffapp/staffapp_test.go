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
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus/stores/hospitalstaffdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus/stores/staffdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus/stores/userdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var errStore = errors.New("store failure")

const (
	// validBody registers a user, the staff record that belongs to it, and the
	// staff's assignment to hospital 7, in one request.
	validBody = `{"username":"somchai","password":"secret123","hospital":7}`

	loginPassword  = "secret123"
	validLoginBody = `{"username":"somchai","password":"secret123","hospital":7}`
)

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

// deps holds the store mocks the App's three businesses are built from.
type deps struct {
	staff         *staffdb.StoreMock
	hospitalStaff *hospitalstaffdb.StoreMock
	user          *userdb.StoreMock
}

func newDeps() deps {
	return deps{
		staff:         staffdb.NewStoreMock(),
		hospitalStaff: hospitalstaffdb.NewStoreMock(),
		user:          userdb.NewStoreMock(),
	}
}

// hashOf is the stored password value a real userbus.Create would produce.
func hashOf(t *testing.T, plain string) string {
	t.Helper()

	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	require.NoError(t, err)

	return string(h)
}

// newLoginDeps wires the happy path of a login: the username resolves to a user
// whose hash matches loginPassword, that user owns staff 42, and staff 42 holds
// an active assignment to hospital 7.
func newLoginDeps(t *testing.T) deps {
	t.Helper()

	d := newDeps()

	d.user.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]userbus.User{{ID: 1, Username: "somchai", Password: hashOf(t, loginPassword)}}, nil).Maybe()

	d.staff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]staffbus.Staff{{
			ID: 42, UserID: 1, EmployeeCode: "EMP-001",
			FirstName: "Somchai", LastName: "Jaidee", IsActive: true,
		}}, nil).Maybe()

	d.hospitalStaff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]hospitalstaffbus.HospitalStaff{{
			ID: 9, HospitalID: 7, StaffID: 42, Role: hospitalstaffbus.RoleRegistrar, IsPrimary: true,
		}}, nil).Maybe()

	return d
}

func newTestRouter(d deps) *gin.Engine {
	return newTestRouterWithTrans(d, nil)
}

func newTestRouterWithTrans(d deps, bgn sqldb.Beginner) *gin.Engine {
	gin.SetMode(gin.TestMode)

	log := logger.New(io.Discard, logger.LevelError, "test")
	api := staffapp.NewApp(
		hospitalstaffbus.NewBusiness(log, d.hospitalStaff),
		staffbus.NewBusiness(log, d.staff),
		userbus.NewBusiness(log, d.user),
	)

	trans := []gin.HandlerFunc{}
	if bgn != nil {
		trans = append(trans, mid.BeginCommitRollback(log, bgn))
	}

	r := gin.New()
	g := r.Group("/api/v1")

	g.POST("/staff/create", append(trans, api.Create)...)
	g.POST("/staff/login", api.Login)
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

func TestApp_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newLoginDeps(t)

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)

		var got staffapp.Authentication
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.True(t, got.Authenticate)
		assert.Equal(t, 1, got.UserID)
		assert.Equal(t, 42, got.StaffID)
		assert.Equal(t, "EMP-001", got.EmployeeCode)
		assert.Equal(t, "Somchai", got.FirstName)
		assert.Equal(t, 7, got.HospitalID)
		assert.Equal(t, hospitalstaffbus.RoleRegistrar, got.Role)
	})

	t.Run("looks the account up by username and the staff by user id", func(t *testing.T) {
		d := newDeps()

		d.user.On("Query", mock.Anything, mock.MatchedBy(func(f userbus.QueryFilter) bool {
			return f.Username != nil && *f.Username == "somchai"
		}), page.MustParse(1, 1), userbus.DefaultOrderBy).
			Return([]userbus.User{{ID: 1, Username: "somchai", Password: hashOf(t, loginPassword)}}, nil).Once()

		d.staff.On("Query", mock.Anything, mock.MatchedBy(func(f staffbus.QueryFilter) bool {
			return f.UserID != nil && *f.UserID == 1
		}), page.MustParse(1, 1), staffbus.DefaultOrderBy).
			Return([]staffbus.Staff{{ID: 42, UserID: 1, IsActive: true}}, nil).Once()

		d.hospitalStaff.On("Query", mock.Anything, mock.MatchedBy(func(f hospitalstaffbus.QueryFilter) bool {
			return f.StaffID != nil && *f.StaffID == 42 &&
				f.HospitalID != nil && *f.HospitalID == 7 &&
				f.Active != nil && *f.Active
		}), page.MustParse(1, 1), hospitalstaffbus.DefaultOrderBy).
			Return([]hospitalstaffbus.HospitalStaff{{ID: 9, HospitalID: 7, StaffID: 42, Role: hospitalstaffbus.RoleNurse}}, nil).Once()

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusOK, w.Code)
		d.user.AssertExpectations(t)
		d.staff.AssertExpectations(t)
		d.hospitalStaff.AssertExpectations(t)
	})

	t.Run("wrong password maps to 401", func(t *testing.T) {
		d := newLoginDeps(t)

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login",
			`{"username":"somchai","password":"wrong-password","hospital":7}`)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Unauthorized", env.Error.Code)
		// The staff lookup must not run once the credentials are rejected.
		d.staff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("unknown username maps to 401 with the same message as a wrong password", func(t *testing.T) {
		d := newDeps()
		d.user.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]userbus.User{}, nil).Once()

		w, unknown := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)
		require.Equal(t, http.StatusUnauthorized, w.Code)

		w2, wrongPass := do(t, newTestRouter(newLoginDeps(t)), http.MethodPost, "/api/v1/staff/login",
			`{"username":"somchai","password":"wrong-password","hospital":7}`)
		require.Equal(t, http.StatusUnauthorized, w2.Code)

		// An unknown username must not be distinguishable from a wrong password.
		require.NotNil(t, unknown.Error)
		require.NotNil(t, wrongPass.Error)
		assert.Equal(t, wrongPass.Error, unknown.Error)
	})

	t.Run("a user with no staff record maps to 401", func(t *testing.T) {
		d := newLoginDeps(t)
		d.staff = staffdb.NewStoreMock()
		d.staff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]staffbus.Staff{}, nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Unauthorized", env.Error.Code)
		d.hospitalStaff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("an inactive staff account maps to 403", func(t *testing.T) {
		d := newLoginDeps(t)
		d.staff = staffdb.NewStoreMock()
		d.staff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]staffbus.Staff{{ID: 42, UserID: 1, IsActive: false}}, nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusForbidden, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Staff Inactive", env.Error.Code)
		d.hospitalStaff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a hospital the staff is not assigned to maps to 403", func(t *testing.T) {
		d := newLoginDeps(t)
		d.hospitalStaff = hospitalstaffdb.NewStoreMock()
		d.hospitalStaff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]hospitalstaffbus.HospitalStaff{}, nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login",
			`{"username":"somchai","password":"secret123","hospital":99}`)

		require.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, env.Success)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Not Allowed", env.Error.Code)
		d.hospitalStaff.AssertExpectations(t)
	})

	t.Run("credentials are never echoed back", func(t *testing.T) {
		d := newLoginDeps(t)

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), loginPassword)
		assert.NotContains(t, w.Body.String(), "password")
	})

	t.Run("malformed body", func(t *testing.T) {
		d := newDeps()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", `{"username":`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Argument", env.Error.Code)
		d.user.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("required fields", func(t *testing.T) {
		cases := map[string]string{
			"missing username":  `{"password":"secret123","hospital":7}`,
			"missing password":  `{"username":"somchai","hospital":7}`,
			"missing hospital":  `{"username":"somchai","password":"secret123"}`,
			"zero hospital":     `{"username":"somchai","password":"secret123","hospital":0}`,
			"negative hospital": `{"username":"somchai","password":"secret123","hospital":-1}`,
			"empty body":        ``,
		}

		for name, body := range cases {
			t.Run(name, func(t *testing.T) {
				d := newDeps()

				w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", body)

				require.Equal(t, http.StatusBadRequest, w.Code)
				d.user.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		d := newDeps()
		d.user.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, env.Success)
		d.user.AssertExpectations(t)
	})

	t.Run("a staff store error maps to 500 rather than 401", func(t *testing.T) {
		d := newLoginDeps(t)
		d.staff = staffdb.NewStoreMock()
		d.staff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("a hospital staff store error maps to 500 rather than 403", func(t *testing.T) {
		d := newLoginDeps(t)
		d.hospitalStaff = hospitalstaffdb.NewStoreMock()
		d.hospitalStaff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/login", validLoginBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestApp_Create(t *testing.T) {
	// newCreateDeps wires the happy path of a registration.
	newCreateDeps := func() deps {
		d := newDeps()

		d.user.On("Create", mock.Anything, mock.MatchedBy(func(u userbus.User) bool {
			return u.Username == "somchai" &&
				bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("secret123")) == nil
		})).Return(1, nil).Once()

		d.staff.On("Create", mock.Anything, mock.MatchedBy(func(s staffbus.Staff) bool {
			// The body carries no staff details, so employee_code is seeded
			// from the username and the names are left for a later update.
			return s.UserID == 1 &&
				s.EmployeeCode == "somchai" &&
				s.FirstName == "" &&
				s.LastName == "" &&
				s.IsActive
		})).Return(42, nil).Once()

		d.hospitalStaff.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.StaffID == 42 && hs.HospitalID == 7 && hs.Role == hospitalstaffbus.RoleRegistrar
		})).Return(9, nil).Once()

		return d
	}

	t.Run("success", func(t *testing.T) {
		d := newCreateDeps()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)

		var got staffapp.Registration
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 42, got.Staff.ID)
		assert.Equal(t, 1, got.Staff.UserID)
		assert.Equal(t, "somchai", got.Staff.EmployeeCode)
		assert.True(t, got.Staff.IsActive)
		assert.NotEmpty(t, got.Staff.CreatedAt)
		assert.Equal(t, 7, got.HospitalID)
		assert.Equal(t, hospitalstaffbus.RoleRegistrar, got.Role)

		d.user.AssertExpectations(t)
		d.staff.AssertExpectations(t)
		d.hospitalStaff.AssertExpectations(t)
	})

	t.Run("the password is hashed and never returned", func(t *testing.T) {
		d := newCreateDeps()

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), "secret123")
		d.user.AssertExpectations(t)
	})

	t.Run("the three writes are chained on the ids of the previous one", func(t *testing.T) {
		d := newDeps()
		d.user.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).Return(11, nil).Once()
		d.staff.On("Create", mock.Anything, mock.MatchedBy(func(s staffbus.Staff) bool {
			return s.UserID == 11
		})).Return(22, nil).Once()
		d.hospitalStaff.On("Create", mock.Anything, mock.MatchedBy(func(hs hospitalstaffbus.HospitalStaff) bool {
			return hs.StaffID == 22 && hs.HospitalID == 7
		})).Return(33, nil).Once()

		w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		d.user.AssertExpectations(t)
		d.staff.AssertExpectations(t)
		d.hospitalStaff.AssertExpectations(t)
	})

	t.Run("malformed body", func(t *testing.T) {
		d := newDeps()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", `{"username":`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Invalid Argument", env.Error.Code)
		d.user.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("required fields", func(t *testing.T) {
		cases := map[string]string{
			"missing username":  `{"password":"secret123","hospital":7}`,
			"short username":    `{"username":"ab","password":"secret123","hospital":7}`,
			"missing password":  `{"username":"somchai","hospital":7}`,
			"short password":    `{"username":"somchai","password":"123","hospital":7}`,
			"missing hospital":  `{"username":"somchai","password":"secret123"}`,
			"zero hospital":     `{"username":"somchai","password":"secret123","hospital":0}`,
			"negative hospital": `{"username":"somchai","password":"secret123","hospital":-1}`,
			"empty body":        ``,
		}

		for name, body := range cases {
			t.Run(name, func(t *testing.T) {
				d := newDeps()

				w, _ := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", body)

				require.Equal(t, http.StatusBadRequest, w.Code)
				d.user.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				d.staff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("a taken username maps to 409 and no staff is written", func(t *testing.T) {
		d := newDeps()
		d.user.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "User Already Exist", env.Error.Code)
		d.staff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		d.hospitalStaff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("a duplicate staff maps to 409 and no assignment is written", func(t *testing.T) {
		d := newDeps()
		d.user.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).Return(1, nil).Once()
		d.staff.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).
			Return(0, sqldb.ErrDBDuplicatedEntry).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusConflict, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Staff Already Exist", env.Error.Code)
		d.hospitalStaff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("an unknown hospital maps to 404", func(t *testing.T) {
		d := newDeps()
		d.user.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).Return(1, nil).Once()
		d.staff.On("Create", mock.Anything, mock.AnythingOfType("staffbus.Staff")).Return(42, nil).Once()
		d.hospitalStaff.On("Create", mock.Anything, mock.AnythingOfType("hospitalstaffbus.HospitalStaff")).
			Return(0, sqldb.ErrDBForeignKeyViolation).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Hospital Not Found", env.Error.Code)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		d := newDeps()
		d.user.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(0, errStore).Once()

		w, env := do(t, newTestRouter(d), http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, env.Success)
		d.user.AssertExpectations(t)
	})

	t.Run("all three writes run on the request transaction", func(t *testing.T) {
		tx := &txMock{}
		tx.On("Commit").Return(nil).Once()
		tx.On("Rollback").Return(nil).Maybe()

		txDeps := newCreateDeps()

		d := newDeps()
		d.user.On("NewWithTx", tx).Return(txDeps.user, nil).Once()
		d.staff.On("NewWithTx", tx).Return(txDeps.staff, nil).Once()
		d.hospitalStaff.On("NewWithTx", tx).Return(txDeps.hospitalStaff, nil).Once()

		r := newTestRouterWithTrans(d, &beginnerMock{tx: tx})
		w, _ := do(t, r, http.MethodPost, "/api/v1/staff/create", validBody)

		require.Equal(t, http.StatusOK, w.Code)
		d.user.AssertExpectations(t)
		d.staff.AssertExpectations(t)
		d.hospitalStaff.AssertExpectations(t)
		txDeps.user.AssertExpectations(t)
		txDeps.staff.AssertExpectations(t)
		txDeps.hospitalStaff.AssertExpectations(t)
		tx.AssertExpectations(t)

		// The writes must go through the tx-scoped stores only.
		d.user.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		d.staff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		d.hospitalStaff.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApp_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{ID: 7, UserID: 1, EmployeeCode: "EMP-001"}, nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got staffapp.Staff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		assert.Equal(t, 7, got.ID)
		assert.Equal(t, "EMP-001", got.EmployeeCode)
		d.staff.AssertExpectations(t)
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		assert.Equal(t, "Staff Not Found", env.Error.Code)
		d.staff.AssertExpectations(t)
	})

	t.Run("unexpected error maps to 500", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).
			Return(staffbus.Staff{}, errStore).Once()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		d.staff.AssertExpectations(t)
	})

	t.Run("non numeric id", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("zero id", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff/0", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})
}

func TestApp_Query(t *testing.T) {
	t.Run("defaults page and order", func(t *testing.T) {
		staffs := []staffbus.Staff{{ID: 1, EmployeeCode: "EMP-001"}, {ID: 2, EmployeeCode: "EMP-002"}}

		d := newDeps()
		d.staff.On("Query", mock.Anything, staffbus.QueryFilter{}, page.MustParse(1, 10), staffbus.DefaultOrderBy).
			Return(staffs, nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff", "")

		require.Equal(t, http.StatusOK, w.Code)

		var got []staffapp.Staff
		require.NoError(t, json.Unmarshal(env.Data, &got))
		require.Len(t, got, 2)
		assert.Equal(t, "EMP-001", got[0].EmployeeCode)

		require.NotNil(t, env.Meta)
		assert.Equal(t, 1, env.Meta.Page)
		assert.Equal(t, 10, env.Meta.PerPage)
		assert.Equal(t, 2, env.Meta.Total)
		d.staff.AssertExpectations(t)
	})

	t.Run("honours filter, paging and ordering", func(t *testing.T) {
		id, userID, pgNum, limit := 5, 3, 2, 20
		orderBy := "created_at,DESC"

		want := staffbus.QueryFilter{ID: &id, UserID: &userID, OrderBy: &orderBy, Page: &pgNum, Limit: &limit}

		d := newDeps()
		d.staff.On("Query", mock.Anything, want, page.MustParse(2, 20), order.NewBy(staffbus.OrderByCreatedAt, order.DESC)).
			Return([]staffbus.Staff{}, nil).Once()

		w, _ := do(t, newTestRouter(d), http.MethodGet,
			"/api/v1/staff?id=5&user_id=3&page=2&limit=20&order_by=created_at,DESC", "")

		require.Equal(t, http.StatusOK, w.Code)
		d.staff.AssertExpectations(t)
	})

	t.Run("unknown order field", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff?order_by=nickname", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("limit above the maximum", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff?limit=1000", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("non numeric query param", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff?user_id=abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		d := newDeps()
		d.staff.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		w, _ := do(t, newTestRouter(d), http.MethodGet, "/api/v1/staff", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		d.staff.AssertExpectations(t)
	})
}

func TestApp_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{ID: 7}, nil).Once()
		d.staff.On("Delete", mock.Anything, 7).Return(nil).Once()

		w, env := do(t, newTestRouter(d), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.True(t, env.Success)
		d.staff.AssertExpectations(t)
	})

	t.Run("unknown staff is not deleted", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{}, sqldb.ErrDBNotFound).Once()

		w, env := do(t, newTestRouter(d), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, env.Error)
		d.staff.AssertExpectations(t)
		d.staff.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("bad id", func(t *testing.T) {
		d := newDeps()

		w, _ := do(t, newTestRouter(d), http.MethodDelete, "/api/v1/staff/abc", "")

		require.Equal(t, http.StatusBadRequest, w.Code)
		d.staff.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("store error maps to 500", func(t *testing.T) {
		d := newDeps()
		d.staff.On("GetByID", mock.Anything, 7).Return(staffbus.Staff{ID: 7}, nil).Once()
		d.staff.On("Delete", mock.Anything, 7).Return(errStore).Once()

		w, _ := do(t, newTestRouter(d), http.MethodDelete, "/api/v1/staff/7", "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		d.staff.AssertExpectations(t)
	})
}
