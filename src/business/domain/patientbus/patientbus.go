package patientbus

import (
	"context"
	"errors"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

var (
	ErrNotFound   = errors.New("error patient not found")
	ErrDuplicate  = errors.New("error patient already exist")
	ErrUnexpected = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, p Patient) (int, error)
	GetByID(ctx context.Context, id int) (Patient, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Patient, error)
	Update(ctx context.Context, p Patient) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, np NewPatient) (Patient, error)
	GetByID(ctx context.Context, id int) (Patient, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Patient, error)
	Delete(ctx context.Context, id int) error
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

func (bus *business) Create(ctx context.Context, np NewPatient) (Patient, error) {
	now := time.Now()
	patient := Patient{
		NationalID:   np.NationalID,
		PassportNo:   np.PassportNo,
		FirstNameTH:  np.FirstNameTH,
		MiddleNameTH: np.MiddleNameTH,
		LastNameTH:   np.LastNameTH,
		FirstNameEN:  np.FirstNameEN,
		MiddleNameEN: np.MiddleNameEN,
		LastNameEN:   np.LastNameEN,
		DateOfBirth:  np.DateOfBirth,
		Gender:       np.Gender,
		Phone:        np.Phone,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := bus.store.Create(ctx, patient)
	if err != nil {
		bus.log.Error(ctx, "create patient error", "err", err)
		if errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			return Patient{}, ErrDuplicate
		}
		return Patient{}, err
	}
	patient.ID = id
	return patient, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (Patient, error) {
	patient, err := bus.store.GetByID(ctx, id)
	if err != nil {
		bus.log.Error(ctx, "error get patient", "err", err)
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return Patient{}, ErrNotFound
		}
		return Patient{}, ErrUnexpected
	}
	return patient, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]Patient, error) {
	patients, err := bus.store.Query(ctx, filter, pg, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query patient", "err", err)
		return nil, ErrUnexpected
	}
	return patients, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete patient", "err", err)
		return ErrUnexpected
	}
	return nil
}
