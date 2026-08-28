package hospitalpatientbus

import "time"

// Set of registration statuses used by the hospital_patients table.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type HospitalPatient struct {
	ID           int
	HospitalID   int
	PatientID    int
	HN           string
	Status       string
	RegisteredAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NewHospitalPatient struct {
	HospitalID   int
	PatientID    int
	HN           string
	Status       string
	RegisteredAt time.Time
}
