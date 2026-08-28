package userbus_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

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

func newTestBusiness(store userbus.Store) userbus.Business {
	log := logger.New(io.Discard, logger.LevelError, "test")
	return userbus.NewBusiness(log, store)
}

func TestBusiness_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		nu := userbus.NewUser{Username: "gopher", Password: "secret"}

		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(u userbus.User) bool {
			return u.Username == nu.Username &&
				u.Password != nu.Password &&
				bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(nu.Password)) == nil &&
				!u.CreatedAt.IsZero() &&
				u.CreatedAt.Equal(u.UpdatedAt)
		})).Return(42, nil).Once()

		usr, err := newTestBusiness(store).Create(context.Background(), nu)

		require.NoError(t, err)
		assert.Equal(t, 42, usr.ID)
		assert.Equal(t, nu.Username, usr.Username)
		assert.False(t, usr.CreatedAt.IsZero())
		assert.Equal(t, usr.CreatedAt, usr.UpdatedAt)
		store.AssertExpectations(t)
	})

	t.Run("password is hashed, never stored or returned in plaintext", func(t *testing.T) {
		nu := userbus.NewUser{Username: "gopher", Password: "secret"}

		var stored userbus.User

		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Run(func(args mock.Arguments) {
				stored = args.Get(1).(userbus.User)
			}).Return(42, nil).Once()

		usr, err := newTestBusiness(store).Create(context.Background(), nu)

		require.NoError(t, err)
		assert.NotEqual(t, nu.Password, stored.Password)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(nu.Password)))
		// The returned user carries the same hash the store was handed.
		assert.Equal(t, stored.Password, usr.Password)
		store.AssertExpectations(t)
	})

	t.Run("hash uses the configured cost", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(42, nil).Once()

		usr, err := newTestBusiness(store).Create(context.Background(),
			userbus.NewUser{Username: "gopher", Password: "secret"})

		require.NoError(t, err)

		cost, err := bcrypt.Cost([]byte(usr.Password))
		require.NoError(t, err)
		assert.Equal(t, bcrypt.DefaultCost, cost)
		store.AssertExpectations(t)
	})

	t.Run("same password hashes differently each time", func(t *testing.T) {
		nu := userbus.NewUser{Username: "gopher", Password: "secret"}

		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(42, nil).Twice()

		bus := newTestBusiness(store)

		first, err := bus.Create(context.Background(), nu)
		require.NoError(t, err)

		second, err := bus.Create(context.Background(), nu)
		require.NoError(t, err)

		// bcrypt salts each hash; identical passwords must not collide.
		assert.NotEqual(t, first.Password, second.Password)
		store.AssertExpectations(t)
	})

	t.Run("empty password is still hashed", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.MatchedBy(func(u userbus.User) bool {
			return u.Password != ""
		})).Return(42, nil).Once()

		// The business does not reject an empty password; that check belongs to
		// the binding rules at the app layer.
		usr, err := newTestBusiness(store).Create(context.Background(),
			userbus.NewUser{Username: "gopher"})

		require.NoError(t, err)
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte("")))
		store.AssertExpectations(t)
	})

	t.Run("password over 72 bytes maps to ErrInvalidPassword", func(t *testing.T) {
		store := userdb.NewStoreMock()

		usr, err := newTestBusiness(store).Create(context.Background(),
			userbus.NewUser{Username: "gopher", Password: strings.Repeat("a", 73)})

		require.ErrorIs(t, err, userbus.ErrInvalidPassword)
		require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
		assert.Equal(t, userbus.User{}, usr)
		store.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("72 byte password is accepted", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(42, nil).Once()

		_, err := newTestBusiness(store).Create(context.Background(),
			userbus.NewUser{Username: "gopher", Password: strings.Repeat("a", 72)})

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("store error returns zero user and passes error through", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Create", mock.Anything, mock.AnythingOfType("userbus.User")).
			Return(0, errStore).Once()

		usr, err := newTestBusiness(store).Create(context.Background(),
			userbus.NewUser{Username: "gopher", Password: "secret"})

		require.ErrorIs(t, err, errStore)
		assert.Equal(t, userbus.User{}, usr)
		store.AssertExpectations(t)
	})
}

func TestBusiness_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := userbus.User{ID: 7, Username: "gopher"}

		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).Return(want, nil).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("no rows maps to ErrNotFound", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(userbus.User{}, sqldb.ErrDBNotFound).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, userbus.ErrNotFound)
		assert.Equal(t, userbus.User{}, got)
		store.AssertExpectations(t)
	})

	t.Run("other error maps to ErrUnexpected", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("GetByID", mock.Anything, 7).
			Return(userbus.User{}, errStore).Once()

		got, err := newTestBusiness(store).GetByID(context.Background(), 7)

		require.ErrorIs(t, err, userbus.ErrUnexpected)
		assert.Equal(t, userbus.User{}, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Query(t *testing.T) {
	filter := userbus.QueryFilter{}
	pg := page.MustParse(1, 10)
	orderBy := order.NewBy(userbus.OrderByID, order.ASC)

	t.Run("success", func(t *testing.T) {
		want := []userbus.User{{ID: 1, Username: "a"}, {ID: 2, Username: "b"}}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).Return(want, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		store.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return([]userbus.User{}, nil).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.NoError(t, err)
		assert.Empty(t, got)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, filter, pg, orderBy).
			Return(nil, errStore).Once()

		got, err := newTestBusiness(store).Query(context.Background(), filter, pg, orderBy)

		require.ErrorIs(t, err, userbus.ErrUnexpected)
		assert.Nil(t, got)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(nil).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.NoError(t, err)
		store.AssertExpectations(t)
	})

	t.Run("store error maps to ErrUnexpected", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Delete", mock.Anything, 7).Return(errStore).Once()

		err := newTestBusiness(store).Delete(context.Background(), 7)

		require.ErrorIs(t, err, userbus.ErrUnexpected)
		store.AssertExpectations(t)
	})
}

func TestBusiness_NewWithTx(t *testing.T) {
	t.Run("success returns business backed by the tx store", func(t *testing.T) {
		tx := &txMock{}
		txStore := userdb.NewStoreMock()

		store := userdb.NewStoreMock()
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

		store := userdb.NewStoreMock()
		store.On("NewWithTx", tx).Return(nil, errStore).Once()

		busTx, err := newTestBusiness(store).NewWithTx(tx)

		require.ErrorIs(t, err, errStore)
		assert.Nil(t, busTx)
		store.AssertExpectations(t)
	})
}

func TestBusiness_Authenticate(t *testing.T) {
	const password = "secret123"

	// hashOf is the stored value a real Create would have produced.
	hashOf := func(t *testing.T, plain string) string {
		t.Helper()
		h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		require.NoError(t, err)
		return string(h)
	}

	// usernameFilter matches a filter that looks the given username up.
	usernameFilter := func(want string) any {
		return mock.MatchedBy(func(f userbus.QueryFilter) bool {
			return f.Username != nil && *f.Username == want && f.ID == nil
		})
	}

	t.Run("success", func(t *testing.T) {
		stored := userbus.User{ID: 42, Username: "gopher", Password: hashOf(t, password)}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, usernameFilter("gopher"), page.MustParse(1, 1), userbus.DefaultOrderBy).
			Return([]userbus.User{stored}, nil).Once()

		usr, err := newTestBusiness(store).Authenticate(context.Background(), "gopher", password)

		require.NoError(t, err)
		assert.Equal(t, 42, usr.ID)
		assert.Equal(t, "gopher", usr.Username)
		store.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		stored := userbus.User{ID: 42, Username: "gopher", Password: hashOf(t, password)}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]userbus.User{stored}, nil).Once()

		usr, err := newTestBusiness(store).Authenticate(context.Background(), "gopher", "not-the-password")

		require.ErrorIs(t, err, userbus.ErrAuthenticationFailure)
		assert.Zero(t, usr)
		store.AssertExpectations(t)
	})

	t.Run("unknown username is indistinguishable from a wrong password", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]userbus.User{}, nil).Once()

		usr, err := newTestBusiness(store).Authenticate(context.Background(), "nobody", password)

		require.ErrorIs(t, err, userbus.ErrAuthenticationFailure)
		assert.Zero(t, usr)
		store.AssertExpectations(t)
	})

	t.Run("a corrupt stored hash does not authenticate", func(t *testing.T) {
		stored := userbus.User{ID: 42, Username: "gopher", Password: "not-a-bcrypt-hash"}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]userbus.User{stored}, nil).Once()

		_, err := newTestBusiness(store).Authenticate(context.Background(), "gopher", password)

		require.ErrorIs(t, err, userbus.ErrAuthenticationFailure)
	})

	t.Run("store error does not leak as an auth failure", func(t *testing.T) {
		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errStore).Once()

		_, err := newTestBusiness(store).Authenticate(context.Background(), "gopher", password)

		require.ErrorIs(t, err, userbus.ErrUnexpected)
		assert.NotErrorIs(t, err, userbus.ErrAuthenticationFailure)
		store.AssertExpectations(t)
	})

	t.Run("the password never travels back to the caller in plaintext", func(t *testing.T) {
		stored := userbus.User{ID: 42, Username: "gopher", Password: hashOf(t, password)}

		store := userdb.NewStoreMock()
		store.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]userbus.User{stored}, nil).Once()

		usr, err := newTestBusiness(store).Authenticate(context.Background(), "gopher", password)

		require.NoError(t, err)
		assert.NotEqual(t, password, usr.Password)
		assert.True(t, strings.HasPrefix(usr.Password, "$2a$"))
	})
}
