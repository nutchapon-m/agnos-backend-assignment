package hospitalpatientapp

import (
	"fmt"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
)

var (
	isoLayout = time.RFC3339
)

type HospitalPatient struct {
	ID           int    `json:"id"`
	HospitalID   int    `json:"hospital_id"`
	PatientID    int    `json:"patient_id"`
	HN           string `json:"hn"`
	Status       string `json:"status"`
	RegisteredAt string `json:"registered_at"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toAppHospitalPatient(hp hospitalpatientbus.HospitalPatient) HospitalPatient {
	return HospitalPatient{
		ID:           hp.ID,
		HospitalID:   hp.HospitalID,
		PatientID:    hp.PatientID,
		HN:           hp.HN,
		Status:       hp.Status,
		RegisteredAt: hp.RegisteredAt.Format(isoLayout),
		CreatedAt:    hp.CreatedAt.Format(isoLayout),
		UpdatedAt:    hp.UpdatedAt.Format(isoLayout),
	}
}

func toAppHospitalPatients(hospitalPatients []hospitalpatientbus.HospitalPatient) []HospitalPatient {
	items := make([]HospitalPatient, len(hospitalPatients))
	for i, hp := range hospitalPatients {
		items[i] = toAppHospitalPatient(hp)
	}
	return items
}

type NewHospitalPatient struct {
	HospitalID   int    `json:"hospital_id" binding:"required,gt=0"`
	PatientID    int    `json:"patient_id" binding:"required,gt=0"`
	HN           string `json:"hn" binding:"required,max=32"`
	Status       string `json:"status" binding:"omitempty,oneof=active inactive"`
	RegisteredAt string `json:"registered_at" binding:"omitempty"`
}

func toNewHospitalPatient(nhp NewHospitalPatient) (hospitalpatientbus.NewHospitalPatient, error) {
	registeredAt, err := parseTimestamp(nhp.RegisteredAt)
	if err != nil {
		return hospitalpatientbus.NewHospitalPatient{}, err
	}

	hospitalPatient := hospitalpatientbus.NewHospitalPatient{
		HospitalID:   nhp.HospitalID,
		PatientID:    nhp.PatientID,
		HN:           nhp.HN,
		Status:       nhp.Status,
		RegisteredAt: registeredAt,
	}
	return hospitalPatient, nil
}

func parseTimestamp(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}

	ts, err := time.Parse(isoLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("registered_at must be in RFC3339 format")
	}
	return ts, nil
}
