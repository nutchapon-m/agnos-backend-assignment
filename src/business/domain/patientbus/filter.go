package patientbus

import "time"

type QueryFilter struct {
	ID          *int
	NationalID  *string
	PassportNo  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	Phone       *string
	Email       *string
	DateOfBirth *time.Time
	OrderBy     *string
	Page        *int
	Limit       *int
}
