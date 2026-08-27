package patientdb

type patient struct {
	ID           string `db:"id"`
	FirstNameTH  string `db:"first_name_th"`
	MiddleNameTH string `db:"middle_name_th"`
	LastNameTH   string `db:"last_name_th"`
	FirstNameEN  string `db:"first_name_en"`
	MiddleNameEN string `db:"middle_name_en"`
	LastNameEN   string `db:"last_name_en"`
	DateOfBirth  string `db:"date_of_birth"`
	PatientHn    string `db:"patient_hn"`
	NationalID   string `db:"national_id"`
	PassportID   string `db:"passport_id"`
	PhoneNumber  string `db:"phone_number"`
	Email        string `db:"email"`
	Gender       string `db:"gender"`
}
