package patientsrv

import "github.com/google/uuid"

type Gender string

const (
	Male   Gender = "M"
	Female Gender = "F"
)

type Patient struct {
	ID           uuid.UUID
	FirstNameTH  string
	MiddleNameTH string
	LastNameTH   string
	FirstNameEN  string
	MiddleNameEN string
	LastNameEN   string
	DateOfBirth  string
	PatientHn    string
	NationalID   string
	PassportID   string
	PhoneNumber  string
	Email        string
	Gender       Gender
}
