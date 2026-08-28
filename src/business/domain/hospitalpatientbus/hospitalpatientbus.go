package hospitalpatientbus

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
	ErrNotFound = errors.New("error hospital patient not found")
	// ErrDuplicate covers both unique indexes: (hospital_id, hn) and
	// (hospital_id, patient_id).
	ErrDuplicate = errors.New("error patient already registered with this hospital")
	// ErrInvalidReference means the hospital or the patient does not exist.
	ErrInvalidReference = errors.New("error hospital or patient does not exist")
	ErrUnexpected       = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, hp HospitalPatient) (int, error)
	GetByID(ctx context.Context, id int) (HospitalPatient, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalPatient, error)
	Update(ctx context.Context, hp HospitalPatient) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, nhp NewHospitalPatient) (HospitalPatient, error)
	GetByID(ctx context.Context, id int) (HospitalPatient, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalPatient, error)
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

func (bus *business) Create(ctx context.Context, nhp NewHospitalPatient) (HospitalPatient, error) {
	now := time.Now()

	status := nhp.Status
	if status == "" {
		status = StatusActive
	}

	registeredAt := nhp.RegisteredAt
	if registeredAt.IsZero() {
		registeredAt = now
	}

	hospitalPatient := HospitalPatient{
		HospitalID:   nhp.HospitalID,
		PatientID:    nhp.PatientID,
		HN:           nhp.HN,
		Status:       status,
		RegisteredAt: registeredAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := bus.store.Create(ctx, hospitalPatient)
	if err != nil {
		bus.log.Error(ctx, "create hospital patient error", "err", err)
		switch {
		case errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			return HospitalPatient{}, ErrDuplicate
		case errors.Is(err, sqldb.ErrDBForeignKeyViolation):
			return HospitalPatient{}, ErrInvalidReference
		}
		return HospitalPatient{}, err
	}
	hospitalPatient.ID = id
	return hospitalPatient, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (HospitalPatient, error) {
	hospitalPatient, err := bus.store.GetByID(ctx, id)
	if err != nil {
		bus.log.Error(ctx, "error get hospital patient", "err", err)
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return HospitalPatient{}, ErrNotFound
		}
		return HospitalPatient{}, ErrUnexpected
	}
	return hospitalPatient, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalPatient, error) {
	hospitalPatients, err := bus.store.Query(ctx, filter, pg, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query hospital patient", "err", err)
		return nil, ErrUnexpected
	}
	return hospitalPatients, nil
}

func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete hospital patient", "err", err)
		return ErrUnexpected
	}
	return nil
}
