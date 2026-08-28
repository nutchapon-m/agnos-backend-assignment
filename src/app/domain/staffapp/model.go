package staffapp

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
)

var (
	isoLayout = time.RFC3339
)

type Staff struct {
	ID           int    `json:"id"`
	UserID       int    `json:"user_id"`
	EmployeeCode string `json:"employee_code"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email,omitempty"`
	LicenseNo    string `json:"license_no,omitempty"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toAppStaff(s staffbus.Staff) Staff {
	return Staff{
		ID:           s.ID,
		UserID:       s.UserID,
		EmployeeCode: s.EmployeeCode,
		FirstName:    s.FirstName,
		LastName:     s.LastName,
		Email:        s.Email,
		LicenseNo:    s.LicenseNo,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt.Format(isoLayout),
		UpdatedAt:    s.UpdatedAt.Format(isoLayout),
	}
}

func toAppStaffs(staffs []staffbus.Staff) []Staff {
	items := make([]Staff, len(staffs))
	for i, s := range staffs {
		items[i] = toAppStaff(s)
	}
	return items
}

type NewStaff struct {
	UserID       int    `json:"user_id" binding:"required,gt=0"`
	EmployeeCode string `json:"employee_code" binding:"required,max=32"`
	FirstName    string `json:"first_name" binding:"required,max=32"`
	LastName     string `json:"last_name" binding:"required,max=32"`
	Email        string `json:"email" binding:"omitempty,email,max=255"`
	LicenseNo    string `json:"license_no" binding:"omitempty,max=64"`
}

func toNewStaff(ns NewStaff) staffbus.NewStaff {
	return staffbus.NewStaff{
		UserID:       ns.UserID,
		EmployeeCode: ns.EmployeeCode,
		FirstName:    ns.FirstName,
		LastName:     ns.LastName,
		Email:        ns.Email,
		LicenseNo:    ns.LicenseNo,
	}
}
