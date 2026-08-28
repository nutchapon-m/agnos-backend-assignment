package hospitaldb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
)

type hospital struct {
	ID           int        `db:"id"`
	Code         string     `db:"code"`
	Name         *string    `db:"name"`
	ProvinceCode *string    `db:"province_code"`
	IsActive     *bool      `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func toDBHospital(h hospitalbus.Hospital) hospital {
	isActive := h.IsActive
	return hospital{
		ID:           h.ID,
		Code:         h.Code,
		Name:         toDBNullString(h.Name),
		ProvinceCode: toDBNullString(h.ProvinceCode),
		IsActive:     &isActive,
		CreatedAt:    h.CreatedAt,
		UpdatedAt:    h.UpdatedAt,
	}
}

func toHospital(h hospital) hospitalbus.Hospital {
	return hospitalbus.Hospital{
		ID:           h.ID,
		Code:         h.Code,
		Name:         toBusString(h.Name),
		ProvinceCode: toBusString(h.ProvinceCode),
		IsActive:     h.IsActive != nil && *h.IsActive,
		CreatedAt:    h.CreatedAt,
		UpdatedAt:    h.UpdatedAt,
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
