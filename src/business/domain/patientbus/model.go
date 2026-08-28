package patientbus

import "time"

// Set of genders accepted by the patients table check constraint.
const (
	GenderMale   = "M"
	GenderFemale = "F"
)

type Patient struct {
	ID           int
	NationalID   string
	PassportNo   string
	FirstNameTH  string
	MiddleNameTH string
	LastNameTH   string
	FirstNameEN  string
	MiddleNameEN string
	LastNameEN   string
	DateOfBirth  time.Time
	Gender       string
	Phone        string
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NewPatient struct {
	NationalID   string
	PassportNo   string
	FirstNameTH  string
	MiddleNameTH string
	LastNameTH   string
	FirstNameEN  string
	MiddleNameEN string
	LastNameEN   string
	DateOfBirth  time.Time
	Gender       string
	Phone        string
	Email        string
}
