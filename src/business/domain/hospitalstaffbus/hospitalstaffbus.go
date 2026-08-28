package hospitalstaffbus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

var (
	ErrNotFound = errors.New("error hospital staff not found")
	// ErrDuplicate covers both unique indexes: one active assignment per
	// (staff_id, hospital_id) and one primary hospital per staff.
	ErrDuplicate = errors.New("error staff already assigned to this hospital")
	// ErrInvalidReference means the hospital or the staff does not exist.
	ErrInvalidReference = errors.New("error hospital or staff does not exist")
	ErrInvalidRole      = errors.New("error invalid role")
	ErrUnexpected       = errors.New("unexpected server error")
)

type Store interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Store, error)
	Create(ctx context.Context, hs HospitalStaff) (int, error)
	GetByID(ctx context.Context, id int) (HospitalStaff, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalStaff, error)
	Update(ctx context.Context, hs HospitalStaff) error
	Delete(ctx context.Context, id int) error
}

type Business interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Business, error)
	Create(ctx context.Context, nhs NewHospitalStaff) (HospitalStaff, error)
	GetByID(ctx context.Context, id int) (HospitalStaff, error)
	Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalStaff, error)
	Delete(ctx context.Context, id int) error
	RegisterStaff(ctx context.Context, hospitalID, staffID int) (HospitalStaff, error)
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

func (bus *business) Create(ctx context.Context, nhs NewHospitalStaff) (HospitalStaff, error) {
	if nhs.Role != "" && !ValidRole(nhs.Role) {
		return HospitalStaff{}, fmt.Errorf("%w: %q", ErrInvalidRole, nhs.Role)
	}

	now := time.Now()

	effectiveFrom := nhs.EffectiveFrom
	if effectiveFrom.IsZero() {
		effectiveFrom = now
	}

	hospitalStaff := HospitalStaff{
		HospitalID:    nhs.HospitalID,
		StaffID:       nhs.StaffID,
		Role:          nhs.Role,
		IsPrimary:     nhs.IsPrimary,
		EffectiveFrom: effectiveFrom,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	id, err := bus.store.Create(ctx, hospitalStaff)
	if err != nil {
		bus.log.Error(ctx, "create hospital staff error", "err", err)
		switch {
		case errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			return HospitalStaff{}, ErrDuplicate
		case errors.Is(err, sqldb.ErrDBForeignKeyViolation):
			return HospitalStaff{}, ErrInvalidReference
		}
		return HospitalStaff{}, err
	}
	hospitalStaff.ID = id
	return hospitalStaff, nil
}

func (bus *business) GetByID(ctx context.Context, id int) (HospitalStaff, error) {
	hospitalStaff, err := bus.store.GetByID(ctx, id)
	if err != nil {
		bus.log.Error(ctx, "error get hospital staff", "err", err)
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return HospitalStaff{}, ErrNotFound
		}
		return HospitalStaff{}, ErrUnexpected
	}
	return hospitalStaff, nil
}

func (bus *business) Query(ctx context.Context, filter QueryFilter, pg page.Page, orderBy order.By) ([]HospitalStaff, error) {
	hospitalStaffs, err := bus.store.Query(ctx, filter, pg, orderBy)
	if err != nil {
		bus.log.Error(ctx, "error query hospital staff", "err", err)
		return nil, ErrUnexpected
	}
	return hospitalStaffs, nil
}

// Delete removes the assignment. The hospital_staffs table has no deleted_at
// column, so this is a hard delete.
func (bus *business) Delete(ctx context.Context, id int) error {
	if err := bus.store.Delete(ctx, id); err != nil {
		bus.log.Error(ctx, "error delete hospital staff", "err", err)
		return ErrUnexpected
	}
	return nil
}

func (bus *business) RegisterStaff(ctx context.Context, hospitalID, staffID int) (HospitalStaff, error) {
	now := time.Now()

	hospitalStaff := HospitalStaff{
		HospitalID:    hospitalID,
		StaffID:       staffID,
		Role:          RoleRegistrar,
		IsPrimary:     true,
		EffectiveFrom: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	id, err := bus.store.Create(ctx, hospitalStaff)
	if err != nil {
		bus.log.Error(ctx, "create hospital staff error", "err", err)
		switch {
		case errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			return HospitalStaff{}, ErrDuplicate
		case errors.Is(err, sqldb.ErrDBForeignKeyViolation):
			return HospitalStaff{}, ErrInvalidReference
		}
		return HospitalStaff{}, err
	}
	hospitalStaff.ID = id
	return hospitalStaff, nil
}
