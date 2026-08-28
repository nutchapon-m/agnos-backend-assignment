package userbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"golang.org/x/crypto/bcrypt"
)

// hashCost is the bcrypt cost applied to stored passwords.
const hashCost = bcrypt.DefaultCost

// noSuchUserHash is a valid bcrypt hash of a value no password can be. It is
// compared against when the username is unknown so that a bad username costs
// the same as a bad password and the two cannot be told apart by timing.
const noSuchUserHash = "$2a$10$/2C891BNdC4heFYhKwseQe6t8jL64PmrnglU5PgPPbT0OFB2Ja5zK"

var (
	ErrNotFound   = errors.New("error user not found")
	ErrDuplicate  = errors.New("error user already exist")
	ErrUnexpected = errors.New("unexpected server error")

	// ErrInvalidPassword means the password could not be hashed. bcrypt rejects anything longer than 72 bytes.
	ErrInvalidPassword = errors.New("error password cannot be hashed")

	// ErrAuthenticationFailure covers both an unknown username and a wrong
	// password. The two are deliberately not distinguished.
	ErrAuthenticationFailure = errors.New("error authentication failed")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, u User) (int, error)
	GetByID(ctx context.Context, id int) (User, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error)
	Update(ctx context.Context, u User) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, nu NewUser) (User, error)
	GetByID(ctx context.Context, id int) (User, error)
	Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error)
	Delete(ctx context.Context, id int) error
	Authenticate(ctx context.Context, username, password string) (User, error)
}

type business struct {
	log   *logger.Logger
	store Store
}

func NewBusiness(log *logger.Logger, store Store) *business {
	return &business{log: log, store: store}
}

func (bus *business) NewWithTx(tx sqldb.CommitRollbacker) (Business, error) {
	storer, err := bus.store.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	business := NewBusiness(bus.log, storer)
	return business, nil
}

// Create stores the user with a bcrypt hash of the password. The plaintext is
// never persisted and never logged.
func (bus *business) Create(ctx context.Context, nu NewUser) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password), hashCost)
	if err != nil {
		bus.log.Error(ctx, "hash password error", "err", err)
		return User{}, fmt.Errorf("%w: %w", ErrInvalidPassword, err)
	}

	now := time.Now()
	user := User{
		Username:  nu.Username,
		Password:  string(hash),
		CreatedAt: now,
		UpdatedAt: now,
	}

	id, err := bus.store.Create(ctx, user)
	if err != nil {
		bus.log.Error(ctx, "create user error", "err", err)
		return User{}, err
	}
	user.ID = id
	return user, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (User, error) {
	user, err := bus.store.GetByID(ctx, id)
	if err != nil {
		if err == sqldb.ErrDBNotFound {
			bus.log.Error(ctx, "error get user", "err", err)
			return User{}, ErrNotFound
		}
		bus.log.Error(ctx, "error get user", "err", err)
		return User{}, ErrUnexpected
	}
	return user, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, p page.Page, orderBy order.By) ([]User, error) {
	users, err := bus.store.Query(ctx, filter, p, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query user", "err", err)
		return nil, ErrUnexpected
	}
	return users, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete user", "err", err)
		return ErrUnexpected
	}
	return nil
}

// Authenticate looks the username up and checks the password against the stored
// hash. An unknown username and a wrong password both come back as
// ErrAuthenticationFailure so that callers cannot probe for valid usernames.
func (bus *business) Authenticate(ctx context.Context, username, password string) (User, error) {
	filter := QueryFilter{Username: &username}

	users, err := bus.store.Query(ctx, filter, page.MustParse(1, 1), DefaultOrderBy)
	if err != nil {
		bus.log.Error(ctx, "error query user", "err", err)
		return User{}, ErrUnexpected
	}

	if len(users) == 0 {
		// Burn the same bcrypt cost as a real comparison would.
		bcrypt.CompareHashAndPassword([]byte(noSuchUserHash), []byte(password))
		bus.log.Info(ctx, "authenticate: unknown username")
		return User{}, ErrAuthenticationFailure
	}

	user := users[0]
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		bus.log.Info(ctx, "authenticate: password mismatch", "user_id", user.ID)
		return User{}, ErrAuthenticationFailure
	}

	return user, nil
}
