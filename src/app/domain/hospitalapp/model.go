package hospitalapp

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
)

var (
	isoLayout = time.RFC3339
)

type Hospital struct {
	ID           int    `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name,omitempty"`
	ProvinceCode string `json:"province_code,omitempty"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toAppHospital(h hospitalbus.Hospital) Hospital {
	return Hospital{
		ID:           h.ID,
		Code:         h.Code,
		Name:         h.Name,
		ProvinceCode: h.ProvinceCode,
		IsActive:     h.IsActive,
		CreatedAt:    h.CreatedAt.Format(isoLayout),
		UpdatedAt:    h.UpdatedAt.Format(isoLayout),
	}
}

func toAppHospitals(hospitals []hospitalbus.Hospital) []Hospital {
	items := make([]Hospital, len(hospitals))
	for i, h := range hospitals {
		items[i] = toAppHospital(h)
	}
	return items
}

type NewHospital struct {
	Code         string `json:"code" binding:"required,max=20"`
	Name         string `json:"name" binding:"omitempty,max=255"`
	ProvinceCode string `json:"province_code" binding:"omitempty,len=2,number"`
}

func toNewHospital(nh NewHospital) hospitalbus.NewHospital {
	return hospitalbus.NewHospital{
		Code:         nh.Code,
		Name:         nh.Name,
		ProvinceCode: nh.ProvinceCode,
	}
}
