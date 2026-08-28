package patientbus

type QueryFilter struct {
	ID         *int
	NationalID *string
	PassportNo *string
	Phone      *string
	OrderBy    *string
	Page       *int
	Limit      *int
}
