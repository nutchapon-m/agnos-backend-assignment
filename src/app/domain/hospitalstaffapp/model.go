package hospitalstaffapp

import (
	"fmt"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
)

var (
	isoLayout  = time.RFC3339
	dateLayout = time.DateOnly
)

type HospitalStaff struct {
	ID            int    `json:"id"`
	HospitalID    int    `json:"hospital_id"`
	StaffID       int    `json:"staff_id"`
	Role          string `json:"role,omitempty"`
	IsPrimary     bool   `json:"is_primary"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toAppHospitalStaff(hs hospitalstaffbus.HospitalStaff) HospitalStaff {
	return HospitalStaff{
		ID:            hs.ID,
		HospitalID:    hs.HospitalID,
		StaffID:       hs.StaffID,
		Role:          hs.Role,
		IsPrimary:     hs.IsPrimary,
		EffectiveFrom: formatDate(hs.EffectiveFrom),
		EffectiveTo:   formatDate(hs.EffectiveTo),
		CreatedAt:     hs.CreatedAt.Format(isoLayout),
		UpdatedAt:     hs.UpdatedAt.Format(isoLayout),
	}
}

func toAppHospitalStaffs(hospitalStaffs []hospitalstaffbus.HospitalStaff) []HospitalStaff {
	items := make([]HospitalStaff, len(hospitalStaffs))
	for i, hs := range hospitalStaffs {
		items[i] = toAppHospitalStaff(hs)
	}
	return items
}

type NewHospitalStaff struct {
	HospitalID    int    `json:"hospital_id" binding:"required,gt=0"`
	StaffID       int    `json:"staff_id" binding:"required,gt=0"`
	Role          string `json:"role" binding:"omitempty,oneof=doctor nurse registrar admin"`
	IsPrimary     bool   `json:"is_primary"`
	EffectiveFrom string `json:"effective_from" binding:"omitempty,datetime=2006-01-02"`
}

func toNewHospitalStaff(nhs NewHospitalStaff) (hospitalstaffbus.NewHospitalStaff, error) {
	effectiveFrom, err := parseDate(nhs.EffectiveFrom)
	if err != nil {
		return hospitalstaffbus.NewHospitalStaff{}, err
	}

	hospitalStaff := hospitalstaffbus.NewHospitalStaff{
		HospitalID:    nhs.HospitalID,
		StaffID:       nhs.StaffID,
		Role:          nhs.Role,
		IsPrimary:     nhs.IsPrimary,
		EffectiveFrom: effectiveFrom,
	}
	return hospitalStaff, nil
}

func formatDate(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.Format(dateLayout)
}

func parseDate(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}

	d, err := time.Parse(dateLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must be in %s format", dateLayout)
	}
	return d, nil
}
