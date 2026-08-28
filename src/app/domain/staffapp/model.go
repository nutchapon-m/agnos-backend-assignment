package staffapp

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
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

// NewRegistration is the POST /staff/create body. One request creates the
// login user, the staff record that belongs to it, and the staff's assignment
// to a hospital.
type NewRegistration struct {
	Username   string `json:"username" binding:"required,min=3,max=255"`
	Password   string `json:"password" binding:"required,min=6"`
	HospitalID int    `json:"hospital" binding:"required,gt=0"`
}

func toNewUser(nr NewRegistration) userbus.NewUser {
	return userbus.NewUser{
		Username: nr.Username,
		Password: nr.Password,
	}
}

// toNewStaff builds the staff record for a freshly created user. The request
// carries no staff details, so employee_code is seeded from the username: the
// column is NOT NULL under a unique index, and username is the only unique
// value the request supplies. The names are left blank for a later update.
func toNewStaff(nr NewRegistration, userID int) staffbus.NewStaff {
	return staffbus.NewStaff{
		UserID:       userID,
		EmployeeCode: nr.Username,
	}
}

// Registration is what the create flow returns. The user is represented by
// Staff.UserID; the password never leaves the business layer.
type Registration struct {
	Staff      Staff  `json:"staff"`
	HospitalID int    `json:"hospital_id"`
	Role       string `json:"role"`
}

func toRegistration(s staffbus.Staff, hs hospitalstaffbus.HospitalStaff) Registration {
	return Registration{
		Staff:      toAppStaff(s),
		HospitalID: hs.HospitalID,
		Role:       hs.Role,
	}
}

// LoginStaff is the POST /staff/login body. A staff member logs in against one
// hospital: the credentials alone are not enough, the account must also hold an
// active assignment to that hospital.
type LoginStaff struct {
	Username   string `json:"username" binding:"required,min=3,max=255"`
	Password   string `json:"password" binding:"required,min=6"`
	HospitalID int    `json:"hospital" binding:"required,gt=0"`
}

// Authentication is what a successful login returns. It carries the identity
// the caller authenticated as and the hospital context it is valid for.
type Authentication struct {
	Authenticate bool   `json:"authenticate"`
	UserID       int    `json:"user_id"`
	StaffID      int    `json:"staff_id"`
	EmployeeCode string `json:"employee_code"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	HospitalID   int    `json:"hospital_id"`
	Role         string `json:"role"`
}

func toAuthentication(s staffbus.Staff, hs hospitalstaffbus.HospitalStaff) Authentication {
	return Authentication{
		Authenticate: true,
		UserID:       s.UserID,
		StaffID:      s.ID,
		EmployeeCode: s.EmployeeCode,
		FirstName:    s.FirstName,
		LastName:     s.LastName,
		HospitalID:   hs.HospitalID,
		Role:         hs.Role,
	}
}
