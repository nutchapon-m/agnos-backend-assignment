package patientdb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
)

type patient struct {
	ID           int        `db:"id"`
	NationalID   *string    `db:"national_id"`
	PassportNo   *string    `db:"passport_no"`
	FirstNameTH  *string    `db:"first_name_th"`
	MiddleNameTH *string    `db:"middle_name_th"`
	LastNameTH   *string    `db:"last_name_th"`
	FirstNameEN  *string    `db:"first_name_en"`
	MiddleNameEN *string    `db:"middle_name_en"`
	LastNameEN   *string    `db:"last_name_en"`
	DateOfBirth  *time.Time `db:"date_of_birth"`
	Gender       *string    `db:"gender"`
	Phone        *string    `db:"phone"`
	Email        *string    `db:"email"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func toDBPatient(p patientbus.Patient) patient {
	return patient{
		ID:           p.ID,
		NationalID:   toDBNullString(p.NationalID),
		PassportNo:   toDBNullString(p.PassportNo),
		FirstNameTH:  toDBNullString(p.FirstNameTH),
		MiddleNameTH: toDBNullString(p.MiddleNameTH),
		LastNameTH:   toDBNullString(p.LastNameTH),
		FirstNameEN:  toDBNullString(p.FirstNameEN),
		MiddleNameEN: toDBNullString(p.MiddleNameEN),
		LastNameEN:   toDBNullString(p.LastNameEN),
		DateOfBirth:  toDBNullTime(p.DateOfBirth),
		Gender:       toDBNullString(p.Gender),
		Phone:        toDBNullString(p.Phone),
		Email:        toDBNullString(p.Email),
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func toPatient(p patient) patientbus.Patient {
	return patientbus.Patient{
		ID:           p.ID,
		NationalID:   toBusString(p.NationalID),
		PassportNo:   toBusString(p.PassportNo),
		FirstNameTH:  toBusString(p.FirstNameTH),
		MiddleNameTH: toBusString(p.MiddleNameTH),
		LastNameTH:   toBusString(p.LastNameTH),
		FirstNameEN:  toBusString(p.FirstNameEN),
		MiddleNameEN: toBusString(p.MiddleNameEN),
		LastNameEN:   toBusString(p.LastNameEN),
		DateOfBirth:  toBusTime(p.DateOfBirth),
		Gender:       toBusString(p.Gender),
		Phone:        toBusString(p.Phone),
		Email:        toBusString(p.Email),
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// toDBNullString keeps optional columns NULL instead of empty so the partial
// unique index on national_id doesn't collide on blank values.
func toDBNullString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func toBusString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func toDBNullTime(v time.Time) *time.Time {
	if v.IsZero() {
		return nil
	}
	return &v
}

func toBusTime(v *time.Time) time.Time {
	if v == nil {
		return time.Time{}
	}
	return *v
}
