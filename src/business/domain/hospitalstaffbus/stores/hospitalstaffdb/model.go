package hospitalstaffdb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
)

type hospitalStaff struct {
	ID            int        `db:"id"`
	HospitalID    int        `db:"hospital_id"`
	StaffID       int        `db:"staff_id"`
	Role          *string    `db:"role"`
	IsPrimary     bool       `db:"is_primary"`
	EffectiveFrom time.Time  `db:"effective_from"`
	EffectiveTo   *time.Time `db:"effective_to"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

func toDBHospitalStaff(hs hospitalstaffbus.HospitalStaff) hospitalStaff {
	return hospitalStaff{
		ID:            hs.ID,
		HospitalID:    hs.HospitalID,
		StaffID:       hs.StaffID,
		Role:          toDBNullString(hs.Role),
		IsPrimary:     hs.IsPrimary,
		EffectiveFrom: hs.EffectiveFrom,
		EffectiveTo:   toDBNullTime(hs.EffectiveTo),
		CreatedAt:     hs.CreatedAt,
		UpdatedAt:     hs.UpdatedAt,
	}
}

func toHospitalStaff(hs hospitalStaff) hospitalstaffbus.HospitalStaff {
	return hospitalstaffbus.HospitalStaff{
		ID:            hs.ID,
		HospitalID:    hs.HospitalID,
		StaffID:       hs.StaffID,
		Role:          toBusString(hs.Role),
		IsPrimary:     hs.IsPrimary,
		EffectiveFrom: hs.EffectiveFrom,
		EffectiveTo:   toBusTime(hs.EffectiveTo),
		CreatedAt:     hs.CreatedAt,
		UpdatedAt:     hs.UpdatedAt,
	}
}

func toDBNullString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func toBusString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// toDBNullTime keeps an open ended assignment NULL so the partial unique
// indexes on effective_to IS NULL keep working.
func toDBNullTime(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	return &v
}

func toBusTime(v *time.Time) time.Time {
	if v == nil {
		return time.Time{}
	}
	return *v
}
