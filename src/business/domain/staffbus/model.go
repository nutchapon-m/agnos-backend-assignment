package staffbus

import "time"

type Staff struct {
	ID           int
	UserID       int
	EmployeeCode string
	FirstName    string
	LastName     string
	Email        string
	LicenseNo    string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NewStaff struct {
	UserID       int
	EmployeeCode string
	FirstName    string
	LastName     string
	Email        string
	LicenseNo    string
}
