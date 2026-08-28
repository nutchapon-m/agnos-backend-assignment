package staffdb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
)

type staff struct {
	ID           int        `db:"id"`
	UserID       int        `db:"user_id"`
	EmployeeCode string     `db:"employee_code"`
	FirstName    string     `db:"first_name"`
	LastName     string     `db:"last_name"`
	Email        *string    `db:"email"`
	LicenseNo    *string    `db:"license_no"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func toDBStaff(s staffbus.Staff) staff {
	return staff{
		ID:           s.ID,
		UserID:       s.UserID,
		EmployeeCode: s.EmployeeCode,
		FirstName:    s.FirstName,
		LastName:     s.LastName,
		Email:        toDBNullString(s.Email),
		LicenseNo:    toDBNullString(s.LicenseNo),
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func toStaff(s staff) staffbus.Staff {
	return staffbus.Staff{
		ID:           s.ID,
		UserID:       s.UserID,
		EmployeeCode: s.EmployeeCode,
		FirstName:    s.FirstName,
		LastName:     s.LastName,
		Email:        toBusString(s.Email),
		LicenseNo:    toBusString(s.LicenseNo),
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// toDBNullString keeps optional columns NULL instead of empty so the partial
// unique indexes (employee_code, lower(email)) don't collide on blank values.
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
