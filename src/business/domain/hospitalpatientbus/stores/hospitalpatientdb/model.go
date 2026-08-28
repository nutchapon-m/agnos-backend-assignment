package hospitalpatientdb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
)

type hospitalPatient struct {
	ID           int        `db:"id"`
	HospitalID   int        `db:"hospital_id"`
	PatientID    int        `db:"patient_id"`
	HN           string     `db:"hn"`
	Status       string     `db:"status"`
	RegisteredAt time.Time  `db:"registered_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func toDBHospitalPatient(hp hospitalpatientbus.HospitalPatient) hospitalPatient {
	return hospitalPatient{
		ID:           hp.ID,
		HospitalID:   hp.HospitalID,
		PatientID:    hp.PatientID,
		HN:           hp.HN,
		Status:       hp.Status,
		RegisteredAt: hp.RegisteredAt,
		CreatedAt:    hp.CreatedAt,
		UpdatedAt:    hp.UpdatedAt,
	}
}

func toHospitalPatient(hp hospitalPatient) hospitalpatientbus.HospitalPatient {
	return hospitalpatientbus.HospitalPatient{
		ID:           hp.ID,
		HospitalID:   hp.HospitalID,
		PatientID:    hp.PatientID,
		HN:           hp.HN,
		Status:       hp.Status,
		RegisteredAt: hp.RegisteredAt,
		CreatedAt:    hp.CreatedAt,
		UpdatedAt:    hp.UpdatedAt,
	}
}
