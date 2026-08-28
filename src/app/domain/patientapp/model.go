package patientapp

import (
	"fmt"
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
)

var (
	isoLayout  = time.RFC3339
	dateLayout = time.DateOnly
)

type Patient struct {
	ID           int    `json:"id"`
	NationalID   string `json:"national_id,omitempty"`
	PassportNo   string `json:"passport_no,omitempty"`
	FirstNameTH  string `json:"first_name_th,omitempty"`
	MiddleNameTH string `json:"middle_name_th,omitempty"`
	LastNameTH   string `json:"last_name_th,omitempty"`
	FirstNameEN  string `json:"first_name_en,omitempty"`
	MiddleNameEN string `json:"middle_name_en,omitempty"`
	LastNameEN   string `json:"last_name_en,omitempty"`
	DateOfBirth  string `json:"date_of_birth,omitempty"`
	Gender       string `json:"gender,omitempty"`
	Phone        string `json:"phone,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toAppPatient(p patientbus.Patient) Patient {
	return Patient{
		ID:           p.ID,
		NationalID:   p.NationalID,
		PassportNo:   p.PassportNo,
		FirstNameTH:  p.FirstNameTH,
		MiddleNameTH: p.MiddleNameTH,
		LastNameTH:   p.LastNameTH,
		FirstNameEN:  p.FirstNameEN,
		MiddleNameEN: p.MiddleNameEN,
		LastNameEN:   p.LastNameEN,
		DateOfBirth:  formatDate(p.DateOfBirth),
		Gender:       p.Gender,
		Phone:        p.Phone,
		CreatedAt:    p.CreatedAt.Format(isoLayout),
		UpdatedAt:    p.UpdatedAt.Format(isoLayout),
	}
}

func toAppPatients(patients []patientbus.Patient) []Patient {
	items := make([]Patient, len(patients))
	for i, p := range patients {
		items[i] = toAppPatient(p)
	}
	return items
}

type NewPatient struct {
	NationalID   string `json:"national_id" binding:"omitempty,len=13,number"`
	PassportNo   string `json:"passport_no" binding:"omitempty,max=32"`
	FirstNameTH  string `json:"first_name_th" binding:"required,max=100"`
	MiddleNameTH string `json:"middle_name_th" binding:"omitempty,max=100"`
	LastNameTH   string `json:"last_name_th" binding:"required,max=100"`
	FirstNameEN  string `json:"first_name_en" binding:"omitempty,max=100"`
	MiddleNameEN string `json:"middle_name_en" binding:"omitempty,max=100"`
	LastNameEN   string `json:"last_name_en" binding:"omitempty,max=100"`
	DateOfBirth  string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	Gender       string `json:"gender" binding:"omitempty,oneof=M F"`
	Phone        string `json:"phone" binding:"omitempty,max=32"`
}

func toNewPatient(np NewPatient) (patientbus.NewPatient, error) {
	dob, err := parseDate(np.DateOfBirth)
	if err != nil {
		return patientbus.NewPatient{}, err
	}

	patient := patientbus.NewPatient{
		NationalID:   np.NationalID,
		PassportNo:   np.PassportNo,
		FirstNameTH:  np.FirstNameTH,
		MiddleNameTH: np.MiddleNameTH,
		LastNameTH:   np.LastNameTH,
		FirstNameEN:  np.FirstNameEN,
		MiddleNameEN: np.MiddleNameEN,
		LastNameEN:   np.LastNameEN,
		DateOfBirth:  dob,
		Gender:       np.Gender,
		Phone:        np.Phone,
	}
	return patient, nil
}

func formatDate(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.Format(dateLayout)
}

func parseDate(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}

	d, err := time.Parse(dateLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must be in %s format", dateLayout)
	}
	return d, nil
}
