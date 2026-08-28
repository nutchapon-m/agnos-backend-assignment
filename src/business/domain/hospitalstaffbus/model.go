package hospitalstaffbus

import "time"

// Set of roles accepted by the hospital_staffs table check constraint.
const (
	RoleDoctor    = "doctor"
	RoleNurse     = "nurse"
	RoleRegistrar = "registrar"
	RoleAdmin     = "admin"
)

var roles = map[string]struct{}{
	RoleDoctor:    {},
	RoleNurse:     {},
	RoleRegistrar: {},
	RoleAdmin:     {},
}

// ValidRole reports whether role is one of the roles the table accepts.
func ValidRole(role string) bool {
	_, exists := roles[role]
	return exists
}

// HospitalStaff is a staff member's assignment to a hospital. An assignment
// with a zero EffectiveTo is still active.
type HospitalStaff struct {
	ID            int
	HospitalID    int
	StaffID       int
	Role          string
	IsPrimary     bool
	EffectiveFrom time.Time
	EffectiveTo   time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type NewHospitalStaff struct {
	HospitalID    int
	StaffID       int
	Role          string
	IsPrimary     bool
	EffectiveFrom time.Time
}
