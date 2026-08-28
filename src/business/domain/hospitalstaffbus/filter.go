package hospitalstaffbus

type QueryFilter struct {
	ID         *int
	HospitalID *int
	StaffID    *int
	Role       *string
	IsPrimary  *bool
	// Active selects assignments that have not ended yet when true, and the
	// ended ones when false.
	Active  *bool
	OrderBy *string
	Page    *int
	Limit   *int
}
