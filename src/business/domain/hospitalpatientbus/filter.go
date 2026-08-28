package hospitalpatientbus

type QueryFilter struct {
	ID         *int
	HospitalID *int
	PatientID  *int
	HN         *string
	Status     *string
	OrderBy    *string
	Page       *int
	Limit      *int
}
